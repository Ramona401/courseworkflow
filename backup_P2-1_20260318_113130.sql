--
-- PostgreSQL database dump
--

\restrict CDZh3WefFup9ueuEeR5tLWOSGdXWjhs67JHEbdyjyPEwIYnAn5yyMb3hlx0pdu8

-- Dumped from database version 16.13 (Ubuntu 16.13-0ubuntu0.24.04.1)
-- Dumped by pg_dump version 16.13 (Ubuntu 16.13-0ubuntu0.24.04.1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: ai_configs; Type: TABLE; Schema: public; Owner: tedna_user
--

CREATE TABLE public.ai_configs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    config_key character varying(50) NOT NULL,
    config_value text NOT NULL,
    description text,
    updated_by uuid,
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.ai_configs OWNER TO tedna_user;

--
-- Name: ai_scene_configs; Type: TABLE; Schema: public; Owner: tedna_user
--

CREATE TABLE public.ai_scene_configs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    scene_code character varying(50) NOT NULL,
    model character varying(100),
    temperature numeric(3,2),
    max_tokens integer,
    system_prompt_id uuid,
    is_active boolean DEFAULT true,
    updated_by uuid,
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.ai_scene_configs OWNER TO tedna_user;

--
-- Name: audit_logs; Type: TABLE; Schema: public; Owner: tedna_user
--

CREATE TABLE public.audit_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid,
    action character varying(100) NOT NULL,
    detail jsonb,
    ip character varying(50),
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.audit_logs OWNER TO tedna_user;

--
-- Name: course_indexes; Type: TABLE; Schema: public; Owner: tedna_user
--

CREATE TABLE public.course_indexes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    course_id uuid NOT NULL,
    index_content text NOT NULL,
    index_hash character varying(64),
    page_count integer,
    total_length integer,
    fetched_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.course_indexes OWNER TO tedna_user;

--
-- Name: courses; Type: TABLE; Schema: public; Owner: tedna_user
--

CREATE TABLE public.courses (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    course_code character varying(20) NOT NULL,
    course_name character varying(200),
    external_module_id integer,
    grade_num integer,
    stage character varying(20),
    semester character varying(20),
    status character varying(20) DEFAULT 'active'::character varying,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.courses OWNER TO tedna_user;

--
-- Name: eval_rounds; Type: TABLE; Schema: public; Owner: tedna_user
--

CREATE TABLE public.eval_rounds (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    pipeline_id uuid NOT NULL,
    round_number integer NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying,
    output text,
    score_total numeric(4,2),
    score_e1 numeric(4,2),
    score_e2 numeric(4,2),
    score_e3 numeric(4,2),
    score_e4 numeric(4,2),
    dimensions jsonb,
    model_used character varying(100),
    tokens_used integer DEFAULT 0,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.eval_rounds OWNER TO tedna_user;

--
-- Name: external_data_configs; Type: TABLE; Schema: public; Owner: tedna_user
--

CREATE TABLE public.external_data_configs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    config_key character varying(50) NOT NULL,
    config_value text NOT NULL,
    description text,
    updated_by uuid,
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.external_data_configs OWNER TO tedna_user;

--
-- Name: generated_pages; Type: TABLE; Schema: public; Owner: tedna_user
--

CREATE TABLE public.generated_pages (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    pipeline_id uuid NOT NULL,
    page_number integer NOT NULL,
    page_title character varying(200),
    operation character varying(20) NOT NULL,
    original_html text,
    generated_html text,
    final_html text,
    decision character varying(20) DEFAULT 'pending'::character varying,
    lesson_id integer,
    merge_sources jsonb,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.generated_pages OWNER TO tedna_user;

--
-- Name: pipeline_steps; Type: TABLE; Schema: public; Owner: tedna_user
--

CREATE TABLE public.pipeline_steps (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    pipeline_id uuid NOT NULL,
    step_name character varying(30) NOT NULL,
    step_order integer NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying,
    started_at timestamp with time zone,
    completed_at timestamp with time zone,
    duration_ms bigint DEFAULT 0,
    attempts integer DEFAULT 0,
    step_data jsonb,
    error_message text,
    model_used character varying(100),
    tokens_used integer DEFAULT 0,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.pipeline_steps OWNER TO tedna_user;

--
-- Name: pipelines; Type: TABLE; Schema: public; Owner: tedna_user
--

CREATE TABLE public.pipelines (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    course_code character varying(20) NOT NULL,
    course_name character varying(200),
    external_module_id integer,
    started_by uuid,
    started_at timestamp with time zone DEFAULT now(),
    completed_at timestamp with time zone,
    current_step character varying(30) DEFAULT 'dbCheck'::character varying NOT NULL,
    status character varying(30) DEFAULT 'pending'::character varying,
    auto_mode boolean DEFAULT true,
    error_message text,
    config jsonb,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.pipelines OWNER TO tedna_user;

--
-- Name: prompts; Type: TABLE; Schema: public; Owner: tedna_user
--

CREATE TABLE public.prompts (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    prompt_key character varying(50) NOT NULL,
    content text NOT NULL,
    version integer DEFAULT 1,
    is_current boolean DEFAULT true,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.prompts OWNER TO tedna_user;

--
-- Name: user_course_assignments; Type: TABLE; Schema: public; Owner: tedna_user
--

CREATE TABLE public.user_course_assignments (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    course_code character varying(20) NOT NULL,
    assigned_by uuid,
    assigned_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.user_course_assignments OWNER TO tedna_user;

--
-- Name: users; Type: TABLE; Schema: public; Owner: tedna_user
--

CREATE TABLE public.users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    username character varying(50) NOT NULL,
    display_name character varying(100) NOT NULL,
    password_hash character varying(255) NOT NULL,
    role character varying(20) DEFAULT 'viewer'::character varying NOT NULL,
    status character varying(20) DEFAULT 'active'::character varying,
    last_login_at timestamp with time zone,
    login_count integer DEFAULT 0,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.users OWNER TO tedna_user;

--
-- Data for Name: ai_configs; Type: TABLE DATA; Schema: public; Owner: tedna_user
--

COPY public.ai_configs (id, config_key, config_value, description, updated_by, updated_at) FROM stdin;
d8a66221-58ed-4b10-a220-5de224f374c7	api_base_url	https://oneapi.xingyunlink.com/v1	AI API 基础地址	\N	2026-03-18 06:55:03.547959+08
f1d0cfcd-a0cb-4ab6-9308-153d8c483aa5	default_model	anthropic/claude-sonnet-4-5	默认模型	\N	2026-03-18 06:55:03.547959+08
14316f3b-172c-4157-aec6-04cf12c16074	temperature	0.7	默认温度	\N	2026-03-18 06:55:03.547959+08
0cf41e5a-11a5-4328-99b4-ccbcbc56e9e9	max_tokens	8000	默认最大Token数	\N	2026-03-18 06:55:03.547959+08
bc64001a-ef80-402c-bda4-4a2f469323f6	api_key_enc	PLACEHOLDER_SET_IN_ADMIN	API Key（管理界面配置）	\N	2026-03-18 06:55:03.547959+08
\.


--
-- Data for Name: ai_scene_configs; Type: TABLE DATA; Schema: public; Owner: tedna_user
--

COPY public.ai_scene_configs (id, scene_code, model, temperature, max_tokens, system_prompt_id, is_active, updated_by, updated_at) FROM stdin;
2f437a00-5104-49ad-a697-62fb4310518f	scanner	anthropic/claude-sonnet-4-5	0.30	8000	\N	t	\N	2026-03-18 06:55:03.548929+08
5c862abc-ceb4-410e-a750-9b43b4c2b3f5	evaluator	anthropic/claude-sonnet-4-5	0.50	8000	\N	t	\N	2026-03-18 06:55:03.548929+08
e6bb1b52-60c5-4002-a0e7-a9a4e2140840	meta	anthropic/claude-sonnet-4-5	0.30	12000	\N	t	\N	2026-03-18 06:55:03.548929+08
c844c022-8eba-44d3-802f-cef259c52a26	translator	anthropic/claude-sonnet-4-5	0.30	8000	\N	t	\N	2026-03-18 06:55:03.548929+08
5c6a7c0f-5404-4ae9-bc29-e3741960fb9e	reviewer	anthropic/claude-sonnet-4-5	0.30	4000	\N	t	\N	2026-03-18 06:55:03.548929+08
9f756c86-de2d-4290-a019-5ca7bd6f4628	generator	anthropic/claude-sonnet-4-5	0.50	12000	\N	t	\N	2026-03-18 06:55:03.548929+08
\.


--
-- Data for Name: audit_logs; Type: TABLE DATA; Schema: public; Owner: tedna_user
--

COPY public.audit_logs (id, user_id, action, detail, ip, created_at) FROM stdin;
\.


--
-- Data for Name: course_indexes; Type: TABLE DATA; Schema: public; Owner: tedna_user
--

COPY public.course_indexes (id, course_id, index_content, index_hash, page_count, total_length, fetched_at) FROM stdin;
\.


--
-- Data for Name: courses; Type: TABLE DATA; Schema: public; Owner: tedna_user
--

COPY public.courses (id, course_code, course_name, external_module_id, grade_num, stage, semester, status, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: eval_rounds; Type: TABLE DATA; Schema: public; Owner: tedna_user
--

COPY public.eval_rounds (id, pipeline_id, round_number, status, output, score_total, score_e1, score_e2, score_e3, score_e4, dimensions, model_used, tokens_used, created_at) FROM stdin;
\.


--
-- Data for Name: external_data_configs; Type: TABLE DATA; Schema: public; Owner: tedna_user
--

COPY public.external_data_configs (id, config_key, config_value, description, updated_by, updated_at) FROM stdin;
facc13c9-9ba6-4469-b8b4-2b60189bee46	oss_endpoint	PLACEHOLDER_SET_IN_ADMIN	OSS Endpoint	\N	2026-03-18 06:55:03.551084+08
e10aeede-5a38-48b3-ba7d-4a7c682841c3	oss_bucket	PLACEHOLDER_SET_IN_ADMIN	OSS Bucket名称	\N	2026-03-18 06:55:03.551084+08
6c3e8bc2-dd35-4e11-bbe6-6c54bc40c528	oss_access_key_id	PLACEHOLDER_SET_IN_ADMIN	OSS AccessKey ID	\N	2026-03-18 06:55:03.551084+08
33858944-0cec-4cb3-9a1e-52d72bbcfe89	oss_access_key_enc	PLACEHOLDER_SET_IN_ADMIN	OSS AccessKey Secret（加密）	\N	2026-03-18 06:55:03.551084+08
c758e108-d933-47d9-853b-dc688fb76be3	oss_index_prefix	indexes/	OSS索引文件路径前缀	\N	2026-03-18 06:55:03.551084+08
666d2257-070c-4cc7-b387-8a3bbbf6d95e	oss_html_prefix	lessons/	OSS HTML文件路径前缀	\N	2026-03-18 06:55:03.551084+08
44908dcb-6752-4c43-8776-501220158eb3	push_api_url	PLACEHOLDER_SET_IN_ADMIN	推送回原始服务器的API地址	\N	2026-03-18 06:55:03.551084+08
184d7268-49a7-4bb4-be1c-b9ea8bb9a22e	push_api_token	PLACEHOLDER_SET_IN_ADMIN	推送API认证Token	\N	2026-03-18 06:55:03.551084+08
\.


--
-- Data for Name: generated_pages; Type: TABLE DATA; Schema: public; Owner: tedna_user
--

COPY public.generated_pages (id, pipeline_id, page_number, page_title, operation, original_html, generated_html, final_html, decision, lesson_id, merge_sources, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: pipeline_steps; Type: TABLE DATA; Schema: public; Owner: tedna_user
--

COPY public.pipeline_steps (id, pipeline_id, step_name, step_order, status, started_at, completed_at, duration_ms, attempts, step_data, error_message, model_used, tokens_used, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: pipelines; Type: TABLE DATA; Schema: public; Owner: tedna_user
--

COPY public.pipelines (id, course_code, course_name, external_module_id, started_by, started_at, completed_at, current_step, status, auto_mode, error_message, config, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: prompts; Type: TABLE DATA; Schema: public; Owner: tedna_user
--

COPY public.prompts (id, prompt_key, content, version, is_current, created_by, created_at) FROM stdin;
10000000-0000-0000-0000-000000000001	prompt_a	# Prompt A (Scanner) - 待配置	1	t	00000000-0000-0000-0000-000000000001	2026-03-18 06:55:03.549859+08
10000000-0000-0000-0000-000000000002	prompt_b	# Prompt B (Evaluator) - 待配置	1	t	00000000-0000-0000-0000-000000000001	2026-03-18 06:55:03.549859+08
10000000-0000-0000-0000-000000000003	prompt_c	# Prompt C (Translator) - 待配置	1	t	00000000-0000-0000-0000-000000000001	2026-03-18 06:55:03.549859+08
10000000-0000-0000-0000-000000000004	prompt_d	# Prompt D (Reviewer) - 待配置	1	t	00000000-0000-0000-0000-000000000001	2026-03-18 06:55:03.549859+08
10000000-0000-0000-0000-000000000005	prompt_e	# Prompt E (Meta) - 待配置	1	t	00000000-0000-0000-0000-000000000001	2026-03-18 06:55:03.549859+08
10000000-0000-0000-0000-000000000006	prompt_f	# Prompt F (Generator) - 待配置	1	t	00000000-0000-0000-0000-000000000001	2026-03-18 06:55:03.549859+08
10000000-0000-0000-0000-000000000007	dict	# 解压缩字典 - 待配置	1	t	00000000-0000-0000-0000-000000000001	2026-03-18 06:55:03.549859+08
10000000-0000-0000-0000-000000000008	ability_table	# 能力定位表 - 待配置	1	t	00000000-0000-0000-0000-000000000001	2026-03-18 06:55:03.549859+08
\.


--
-- Data for Name: user_course_assignments; Type: TABLE DATA; Schema: public; Owner: tedna_user
--

COPY public.user_course_assignments (id, user_id, course_code, assigned_by, assigned_at) FROM stdin;
\.


--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: tedna_user
--

COPY public.users (id, username, display_name, password_hash, role, status, last_login_at, login_count, created_at, updated_at) FROM stdin;
00000000-0000-0000-0000-000000000001	admin	系统管理员	$2a$10$BvWU4.Za1.UW2OHYMjxlZ.AhixofyxrCHeaAknFBBCzXxkZSXkWAG	admin	active	2026-03-18 07:54:20.248468+08	7	2026-03-18 06:55:03.546705+08	2026-03-18 07:54:20.248468+08
dd399d34-3c7d-4d81-afe0-722d8ffec707	yingjun	沈老师	$2a$10$GpGvYlqew8Hcqxyq.1JlrO0Uas3asMEwUfuYr1Y8hXFgNOaNK2dve	operator	active	\N	0	2026-03-18 08:08:59.566858+08	2026-03-18 08:09:30.172205+08
\.


--
-- Name: ai_configs ai_configs_config_key_key; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.ai_configs
    ADD CONSTRAINT ai_configs_config_key_key UNIQUE (config_key);


--
-- Name: ai_configs ai_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.ai_configs
    ADD CONSTRAINT ai_configs_pkey PRIMARY KEY (id);


--
-- Name: ai_scene_configs ai_scene_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.ai_scene_configs
    ADD CONSTRAINT ai_scene_configs_pkey PRIMARY KEY (id);


--
-- Name: ai_scene_configs ai_scene_configs_scene_code_key; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.ai_scene_configs
    ADD CONSTRAINT ai_scene_configs_scene_code_key UNIQUE (scene_code);


--
-- Name: audit_logs audit_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.audit_logs
    ADD CONSTRAINT audit_logs_pkey PRIMARY KEY (id);


--
-- Name: course_indexes course_indexes_pkey; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.course_indexes
    ADD CONSTRAINT course_indexes_pkey PRIMARY KEY (id);


--
-- Name: courses courses_course_code_key; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.courses
    ADD CONSTRAINT courses_course_code_key UNIQUE (course_code);


--
-- Name: courses courses_pkey; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.courses
    ADD CONSTRAINT courses_pkey PRIMARY KEY (id);


--
-- Name: eval_rounds eval_rounds_pipeline_id_round_number_key; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.eval_rounds
    ADD CONSTRAINT eval_rounds_pipeline_id_round_number_key UNIQUE (pipeline_id, round_number);


--
-- Name: eval_rounds eval_rounds_pkey; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.eval_rounds
    ADD CONSTRAINT eval_rounds_pkey PRIMARY KEY (id);


--
-- Name: external_data_configs external_data_configs_config_key_key; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.external_data_configs
    ADD CONSTRAINT external_data_configs_config_key_key UNIQUE (config_key);


--
-- Name: external_data_configs external_data_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.external_data_configs
    ADD CONSTRAINT external_data_configs_pkey PRIMARY KEY (id);


--
-- Name: generated_pages generated_pages_pkey; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.generated_pages
    ADD CONSTRAINT generated_pages_pkey PRIMARY KEY (id);


--
-- Name: pipeline_steps pipeline_steps_pipeline_id_step_name_key; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.pipeline_steps
    ADD CONSTRAINT pipeline_steps_pipeline_id_step_name_key UNIQUE (pipeline_id, step_name);


--
-- Name: pipeline_steps pipeline_steps_pkey; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.pipeline_steps
    ADD CONSTRAINT pipeline_steps_pkey PRIMARY KEY (id);


--
-- Name: pipelines pipelines_pkey; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.pipelines
    ADD CONSTRAINT pipelines_pkey PRIMARY KEY (id);


--
-- Name: prompts prompts_pkey; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.prompts
    ADD CONSTRAINT prompts_pkey PRIMARY KEY (id);


--
-- Name: user_course_assignments user_course_assignments_pkey; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.user_course_assignments
    ADD CONSTRAINT user_course_assignments_pkey PRIMARY KEY (id);


--
-- Name: user_course_assignments user_course_assignments_user_id_course_code_key; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.user_course_assignments
    ADD CONSTRAINT user_course_assignments_user_id_course_code_key UNIQUE (user_id, course_code);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: users users_username_key; Type: CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_username_key UNIQUE (username);


--
-- Name: idx_course_indexes_course_id; Type: INDEX; Schema: public; Owner: tedna_user
--

CREATE INDEX idx_course_indexes_course_id ON public.course_indexes USING btree (course_id);


--
-- Name: idx_generated_pages_pipeline_id; Type: INDEX; Schema: public; Owner: tedna_user
--

CREATE INDEX idx_generated_pages_pipeline_id ON public.generated_pages USING btree (pipeline_id);


--
-- Name: idx_pipeline_steps_pipeline_id; Type: INDEX; Schema: public; Owner: tedna_user
--

CREATE INDEX idx_pipeline_steps_pipeline_id ON public.pipeline_steps USING btree (pipeline_id);


--
-- Name: idx_pipelines_course_code; Type: INDEX; Schema: public; Owner: tedna_user
--

CREATE INDEX idx_pipelines_course_code ON public.pipelines USING btree (course_code);


--
-- Name: idx_pipelines_status; Type: INDEX; Schema: public; Owner: tedna_user
--

CREATE INDEX idx_pipelines_status ON public.pipelines USING btree (status);


--
-- Name: idx_prompts_key_current; Type: INDEX; Schema: public; Owner: tedna_user
--

CREATE INDEX idx_prompts_key_current ON public.prompts USING btree (prompt_key, is_current);


--
-- Name: ai_configs ai_configs_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.ai_configs
    ADD CONSTRAINT ai_configs_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id);


--
-- Name: ai_scene_configs ai_scene_configs_system_prompt_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.ai_scene_configs
    ADD CONSTRAINT ai_scene_configs_system_prompt_id_fkey FOREIGN KEY (system_prompt_id) REFERENCES public.prompts(id);


--
-- Name: ai_scene_configs ai_scene_configs_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.ai_scene_configs
    ADD CONSTRAINT ai_scene_configs_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id);


--
-- Name: audit_logs audit_logs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.audit_logs
    ADD CONSTRAINT audit_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: course_indexes course_indexes_course_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.course_indexes
    ADD CONSTRAINT course_indexes_course_id_fkey FOREIGN KEY (course_id) REFERENCES public.courses(id) ON DELETE CASCADE;


--
-- Name: eval_rounds eval_rounds_pipeline_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.eval_rounds
    ADD CONSTRAINT eval_rounds_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipelines(id) ON DELETE CASCADE;


--
-- Name: external_data_configs external_data_configs_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.external_data_configs
    ADD CONSTRAINT external_data_configs_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id);


--
-- Name: generated_pages generated_pages_pipeline_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.generated_pages
    ADD CONSTRAINT generated_pages_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipelines(id) ON DELETE CASCADE;


--
-- Name: pipeline_steps pipeline_steps_pipeline_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.pipeline_steps
    ADD CONSTRAINT pipeline_steps_pipeline_id_fkey FOREIGN KEY (pipeline_id) REFERENCES public.pipelines(id) ON DELETE CASCADE;


--
-- Name: pipelines pipelines_started_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.pipelines
    ADD CONSTRAINT pipelines_started_by_fkey FOREIGN KEY (started_by) REFERENCES public.users(id);


--
-- Name: prompts prompts_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.prompts
    ADD CONSTRAINT prompts_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: user_course_assignments user_course_assignments_assigned_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.user_course_assignments
    ADD CONSTRAINT user_course_assignments_assigned_by_fkey FOREIGN KEY (assigned_by) REFERENCES public.users(id);


--
-- Name: user_course_assignments user_course_assignments_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: tedna_user
--

ALTER TABLE ONLY public.user_course_assignments
    ADD CONSTRAINT user_course_assignments_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: SCHEMA public; Type: ACL; Schema: -; Owner: pg_database_owner
--

GRANT ALL ON SCHEMA public TO tedna_user;


--
-- PostgreSQL database dump complete
--

\unrestrict CDZh3WefFup9ueuEeR5tLWOSGdXWjhs67JHEbdyjyPEwIYnAn5yyMb3hlx0pdu8

