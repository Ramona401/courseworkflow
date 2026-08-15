package services

import (
	"net/http"
	"testing"
)

type identityAccountServiceMutationCase struct {
	name      string
	purpose   string
	platform  IdentityPlatformLink
	back      IdentityBackchannelResult
	wantState string
	wantOp    string

	wantChanged bool
	wantCalls   int

	wantErrCode   string
	wantErrStatus int
}

func TestIdentityAccountLinkServiceMutationBranches(
	t *testing.T,
) {
	tests := []identityAccountServiceMutationCase{
		{
			name: "link exact existing mapping is noop",

			purpose: IdentityFlowPurposeLink,

			platform: IdentityPlatformLink{
				Linked:         true,
				LocalAccountID: identityAccountServiceTestLocalID,
			},

			wantState: "linked",

			wantOp: IdentityBackchannelOperationLink,
		},
		{
			name: "link other local mapping conflicts",

			purpose: IdentityFlowPurposeLink,

			platform: IdentityPlatformLink{
				Linked:         true,
				LocalAccountID: identityAccountServiceTestOtherLocalID,
			},

			wantErrCode: "IDENTITY_LINK_CONFLICT",

			wantErrStatus: http.StatusConflict,
		},
		{
			name: "new link mutates",

			purpose: IdentityFlowPurposeLink,

			platform: IdentityPlatformLink{
				Linked: false,
			},

			back: IdentityBackchannelResult{
				Outcome: "success",
				State:   "linked",
			},

			wantState: "linked",

			wantOp: IdentityBackchannelOperationLink,

			wantChanged: true,

			wantCalls: 1,
		},
		{
			name: "idempotent link replay is unchanged",

			purpose: IdentityFlowPurposeLink,

			platform: IdentityPlatformLink{
				Linked: false,
			},

			back: IdentityBackchannelResult{
				Outcome:          "success",
				State:            "linked",
				IdempotentReplay: true,
			},

			wantState: "linked",

			wantOp: IdentityBackchannelOperationLink,

			wantCalls: 1,
		},
		{
			name: "link stable conflict maps safely",

			purpose: IdentityFlowPurposeLink,

			platform: IdentityPlatformLink{
				Linked: false,
			},

			back: IdentityBackchannelResult{
				Outcome:    "conflict",
				State:      "conflict",
				ReasonCode: "local_account_already_linked",
			},

			wantCalls: 1,

			wantErrCode: "IDENTITY_LINK_CONFLICT",

			wantErrStatus: http.StatusConflict,
		},
		{
			name: "already unlinked is noop",

			purpose: IdentityFlowPurposeUnlink,

			platform: IdentityPlatformLink{
				Linked: false,
			},

			wantState: "unlinked",

			wantOp: IdentityBackchannelOperationUnlink,
		},
		{
			name: "unlink other local mapping conflicts",

			purpose: IdentityFlowPurposeUnlink,

			platform: IdentityPlatformLink{
				Linked:         true,
				LocalAccountID: identityAccountServiceTestOtherLocalID,
			},

			wantErrCode: "IDENTITY_LINK_CONFLICT",

			wantErrStatus: http.StatusConflict,
		},
		{
			name: "exact unlink mutates",

			purpose: IdentityFlowPurposeUnlink,

			platform: IdentityPlatformLink{
				Linked:         true,
				LocalAccountID: identityAccountServiceTestLocalID,
			},

			back: IdentityBackchannelResult{
				Outcome: "success",
				State:   "unlinked",
			},

			wantState: "unlinked",

			wantOp: IdentityBackchannelOperationUnlink,

			wantChanged: true,

			wantCalls: 1,
		},
		{
			name: "idempotent unlink replay is unchanged",

			purpose: IdentityFlowPurposeUnlink,

			platform: IdentityPlatformLink{
				Linked:         true,
				LocalAccountID: identityAccountServiceTestLocalID,
			},

			back: IdentityBackchannelResult{
				Outcome:          "success",
				State:            "unlinked",
				IdempotentReplay: true,
			},

			wantState: "unlinked",

			wantOp: IdentityBackchannelOperationUnlink,

			wantCalls: 1,
		},
		{
			name: "unlink state changed conflict maps safely",

			purpose: IdentityFlowPurposeUnlink,

			platform: IdentityPlatformLink{
				Linked:         true,
				LocalAccountID: identityAccountServiceTestLocalID,
			},

			back: IdentityBackchannelResult{
				Outcome:    "conflict",
				State:      "conflict",
				ReasonCode: "link_not_found",
			},

			wantCalls: 1,

			wantErrCode: "IDENTITY_LINK_STATE_CHANGED",

			wantErrStatus: http.StatusConflict,
		},
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(
			testCase.name,
			func(t *testing.T) {
				fixture :=
					newIdentityAccountServiceTestFixture(
						t,
						IdentityAuthorizationIdentity{
							GlobalPersonID: identityAccountServiceTestGlobalID,
							PlatformLink:   testCase.platform,
						},
						testCase.back,
					)

				result, err :=
					completeIdentityAccountServiceTest(
						t,
						fixture,
						testCase.purpose,
					)

				if testCase.wantErrCode != "" {
					assertIdentityAccountServiceError(
						t,
						err,
						testCase.wantErrStatus,
						testCase.wantErrCode,
					)
				} else {
					if err != nil {
						t.Fatalf(
							"CompleteAuthorization() error = %v",
							err,
						)
					}

					if result.Operation !=
						testCase.wantOp ||
						result.State !=
							testCase.wantState ||
						result.Changed !=
							testCase.wantChanged {
						t.Fatalf(
							"业务结果异常：%+v",
							result,
						)
					}
				}

				if fixture.backchannel.calls !=
					testCase.wantCalls {
					t.Fatalf(
						"Backchannel calls=%d want=%d",
						fixture.backchannel.calls,
						testCase.wantCalls,
					)
				}

				if testCase.wantCalls == 0 {
					return
				}

				expectedOperation :=
					IdentityBackchannelOperationLink

				if testCase.purpose ==
					IdentityFlowPurposeUnlink {
					expectedOperation =
						IdentityBackchannelOperationUnlink
				}

				if fixture.backchannel.operation !=
					expectedOperation {
					t.Fatalf(
						"Mutation operation=%s want=%s",
						fixture.backchannel.operation,
						expectedOperation,
					)
				}

				if fixture.backchannel.globalPersonID !=
					identityAccountServiceTestGlobalID {
					t.Fatalf(
						"Mutation global_person_id=%s",
						fixture.backchannel.globalPersonID,
					)
				}

				if fixture.backchannel.localAccountID !=
					identityAccountServiceTestLocalID {
					t.Fatalf(
						"Mutation local_account_id=%s",
						fixture.backchannel.localAccountID,
					)
				}

				if fixture.backchannel.traceID != "" ||
					fixture.backchannel.idempotencyKey != "" {
					t.Fatal(
						"业务Service不得自行伪造trace或idempotency值",
					)
				}
			},
		)
	}
}
