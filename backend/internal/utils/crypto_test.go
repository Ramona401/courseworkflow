// Package utils 工具函数单元测试
// 测试范围：AES-256-GCM 加密解密核心功能（函数名：EncryptAES / DecryptAES）
package utils

import (
	"strings"
	"testing"
)

// TestEncryptAESBasic 测试基本加密：返回非空密文且与明文不同
func TestEncryptAESBasic(t *testing.T) {
	keyHex := "3132333435363738393031323334353637383930313233343536373839303132" // 32字节hex
	plaintext := "hello world"

	ciphertext, err := EncryptAES(plaintext, keyHex)
	if err != nil {
		t.Fatalf("EncryptAES失败: %v", err)
	}
	if ciphertext == "" {
		t.Fatal("加密结果不应为空")
	}
	if ciphertext == plaintext {
		t.Fatal("加密结果不应与明文相同")
	}
	t.Logf("加密成功，密文: %s...", ciphertext[:min(30, len(ciphertext))])
}

// TestDecryptAESBasic 测试基本解密：加密后解密应还原原文
func TestDecryptAESBasic(t *testing.T) {
	keyHex := "3132333435363738393031323334353637383930313233343536373839303132"
	plaintext := "hello world"

	ciphertext, err := EncryptAES(plaintext, keyHex)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	decrypted, err := DecryptAES(ciphertext, keyHex)
	if err != nil {
		t.Fatalf("DecryptAES失败: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("解密结果错误: 期望 %q, 实际 %q", plaintext, decrypted)
	}
	t.Logf("加解密往返验证通过: %s", decrypted)
}

// TestEncryptAESChinesePlaintext 测试中文明文加解密
func TestEncryptAESChinesePlaintext(t *testing.T) {
	keyHex := "6162636465666768696a6b6c6d6e6f707172737475767778797a303132333435"
	plaintext := "这是一段中文测试内容，包含特殊字符！@#￥%"

	ciphertext, err := EncryptAES(plaintext, keyHex)
	if err != nil {
		t.Fatalf("中文加密失败: %v", err)
	}
	decrypted, err := DecryptAES(ciphertext, keyHex)
	if err != nil {
		t.Fatalf("中文解密失败: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("中文解密结果错误: 期望 %q, 实际 %q", plaintext, decrypted)
	}
	t.Log("中文加解密通过")
}

// TestEncryptAESNonDeterministic 测试随机性：同一明文两次加密结果应不同
func TestEncryptAESNonDeterministic(t *testing.T) {
	keyHex := "3132333435363738393031323334353637383930313233343536373839303132"
	plaintext := "test nonce randomness"

	c1, err1 := EncryptAES(plaintext, keyHex)
	c2, err2 := EncryptAES(plaintext, keyHex)
	if err1 != nil || err2 != nil {
		t.Fatalf("加密失败: %v %v", err1, err2)
	}
	if c1 == c2 {
		t.Fatal("两次加密结果相同，GCM nonce可能未随机化")
	}
	t.Log("随机性验证通过：两次密文不同")
}

// TestDecryptAESWrongKey 测试错误密钥解密：应返回错误
func TestDecryptAESWrongKey(t *testing.T) {
	keyHex := "3132333435363738393031323334353637383930313233343536373839303132"
	wrongKey := "9999999999999999999999999999999999999999999999999999999999999999"
	plaintext := "secret data"

	ciphertext, err := EncryptAES(plaintext, keyHex)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	_, err = DecryptAES(ciphertext, wrongKey)
	if err == nil {
		t.Fatal("错误密钥解密应返回错误，但成功了")
	}
	t.Logf("错误密钥正确返回错误: %v", err)
}

// TestDecryptAESInvalidCiphertext 测试无效密文：应返回错误
func TestDecryptAESInvalidCiphertext(t *testing.T) {
	keyHex := "3132333435363738393031323334353637383930313233343536373839303132"
	_, err := DecryptAES("not_valid_hex!!!", keyHex)
	if err == nil {
		t.Fatal("无效密文应返回解密错误")
	}
	t.Logf("无效密文正确返回错误: %v", err)
}

// TestEncryptAESEmptyPlaintext 测试空字符串加解密
func TestEncryptAESEmptyPlaintext(t *testing.T) {
	keyHex := "3132333435363738393031323334353637383930313233343536373839303132"
	ciphertext, err := EncryptAES("", keyHex)
	if err != nil {
		t.Fatalf("空字符串加密失败: %v", err)
	}
	decrypted, err := DecryptAES(ciphertext, keyHex)
	if err != nil {
		t.Fatalf("空字符串解密失败: %v", err)
	}
	if decrypted != "" {
		t.Fatalf("空字符串解密应返回空，实际: %q", decrypted)
	}
	t.Log("空字符串加解密通过")
}

// TestEncryptAESLongPlaintext 测试长文本加密（模拟OSS/API密钥）
func TestEncryptAESLongPlaintext(t *testing.T) {
	keyHex := "3132333435363738393031323334353637383930313233343536373839303132"
	plaintext := strings.Repeat("ABCDEF", 500) + "末尾特殊内容"

	ciphertext, err := EncryptAES(plaintext, keyHex)
	if err != nil {
		t.Fatalf("长文本加密失败: %v", err)
	}
	decrypted, err := DecryptAES(ciphertext, keyHex)
	if err != nil {
		t.Fatalf("长文本解密失败: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("长文本解密不匹配，长度: 期望%d 实际%d", len(plaintext), len(decrypted))
	}
	t.Logf("长文本(%d字节)加解密通过", len(plaintext))
}

// TestHashPassword 测试密码哈希（bcrypt）
func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("mypassword123")
	if err != nil {
		t.Fatalf("HashPassword失败: %v", err)
	}
	if hash == "" {
		t.Fatal("哈希结果不应为空")
	}
	if hash == "mypassword123" {
		t.Fatal("哈希结果不应与原文相同")
	}
	t.Logf("密码哈希成功，长度: %d", len(hash))
}

// TestCheckPassword 测试密码校验
func TestCheckPassword(t *testing.T) {
	password := "TestPassword456!"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword失败: %v", err)
	}

	// 正确密码应通过
	if !CheckPassword(password, hash) {
		t.Fatal("正确密码校验失败")
	}
	// 错误密码应拒绝
	if CheckPassword("wrongpassword", hash) {
		t.Fatal("错误密码不应通过校验")
	}
	t.Log("密码校验功能验证通过")
}

// TestSHA256Hash 测试SHA256哈希确定性
func TestSHA256Hash(t *testing.T) {
	input := "test content for hashing"
	h1 := SHA256Hash(input)
	h2 := SHA256Hash(input)

	if h1 == "" {
		t.Fatal("SHA256Hash不应返回空")
	}
	if h1 != h2 {
		t.Fatal("SHA256Hash应为确定性函数，两次结果应相同")
	}
	if len(h1) != 64 {
		t.Fatalf("SHA256Hash应为64字符十六进制，实际长度: %d", len(h1))
	}
	t.Logf("SHA256Hash验证通过: %s", h1[:16]+"...")
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
