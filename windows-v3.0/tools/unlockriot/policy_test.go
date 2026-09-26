package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestDetectGPUModel(t *testing.T) {
	tests := []struct {
		name     string
		deviceID uint16
		gpuName  string
		expected GPUModel
	}{
		{"40HX_ByID", DeviceIDCMP40HX, "", GPUModelCMP40HX},
		{"40HX_ByName", 0x9999, "NVIDIA CMP 40HX Graphics", GPUModelCMP40HX},
		{"40HX_ByNameLowercase", 0, "nvidia cmp 40hx", GPUModelCMP40HX},
		{"30HX_ByID", DeviceIDCMP30HX, "", GPUModelCMP30HX},
		{"30HX_ByName", 0, "NVIDIA CMP 30HX", GPUModelCMP30HX},
		{"Unknown_Generic", 0x10DE, "NVIDIA GeForce RTX 3060", GPUModelUnknown},
		{"Empty", 0, "", GPUModelUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := DetectGPUModel(tt.deviceID, tt.gpuName)

			// Assert
			if got != tt.expected {
				t.Errorf("DetectGPUModel(%x, %q) = %v; kỳ vọng %v", tt.deviceID, tt.gpuName, got, tt.expected)
			}
		})
	}
}

func TestEvaluateRiotStatus(t *testing.T) {
	tests := []struct {
		name                   string
		deviceID               uint16
		gpuName                string
		sbOn                   bool
		isWin11                bool
		expectedModel          GPUModel
		expectedTensor         bool
		expectedValorant       bool
		expectedLMHT           bool
		expectedNeedsSigning   bool
		expectedRecommendedSB  string
		expectedBannerKeyword  string
	}{
		{
			name:                  "40HX_Win10_SB_Off_LyTuong",
			deviceID:              DeviceIDCMP40HX,
			gpuName:               "NVIDIA CMP 40HX",
			sbOn:                  false,
			isWin11:               false,
			expectedModel:         GPUModelCMP40HX,
			expectedTensor:        true,
			expectedValorant:      true,
			expectedLMHT:          true,
			expectedNeedsSigning:  false,
			expectedRecommendedSB: "DISABLED",
			expectedBannerKeyword: "TẮT SECURE BOOT (Disabled)",
		},
		{
			name:                  "40HX_Win10_SB_On",
			deviceID:              DeviceIDCMP40HX,
			gpuName:               "NVIDIA CMP 40HX",
			sbOn:                  true,
			isWin11:               false,
			expectedModel:         GPUModelCMP40HX,
			expectedTensor:        true,
			expectedValorant:      true,
			expectedLMHT:          true,
			expectedNeedsSigning:  false,
			expectedRecommendedSB: "DISABLED",
			expectedBannerKeyword: "Vanguard trên Win 10 KHÔNG ép bật",
		},
		{
			name:                  "40HX_Win11_SB_On_CanSigningKey",
			deviceID:              DeviceIDCMP40HX,
			gpuName:               "NVIDIA CMP 40HX",
			sbOn:                  true,
			isWin11:               true,
			expectedModel:         GPUModelCMP40HX,
			expectedTensor:        true,
			expectedValorant:      true,
			expectedLMHT:          true,
			expectedNeedsSigning:  true,
			expectedRecommendedSB: "CUSTOM_ENROLL_ENABLED",
			expectedBannerKeyword: "Tự Động Ký Chữ Ký Số EFI",
		},
		{
			name:                  "40HX_Win11_SB_Off_ChuaKhaDungValorant",
			deviceID:              DeviceIDCMP40HX,
			gpuName:               "NVIDIA CMP 40HX",
			sbOn:                  false,
			isWin11:               true,
			expectedModel:         GPUModelCMP40HX,
			expectedTensor:        true,
			expectedValorant:      false, // Vanguard trên Win 11 bắt buộc bật Secure Boot
			expectedLMHT:          true,  // LMHT không ép Secure Boot
			expectedNeedsSigning:  true,
			expectedRecommendedSB: "CUSTOM_ENROLL_ENABLED",
			expectedBannerKeyword: "Để chơi cả Valorant: Nhấn",
		},
		{
			name:                  "30HX_Win10_SB_On_TuongThichHoanToan",
			deviceID:              DeviceIDCMP30HX,
			gpuName:               "NVIDIA CMP 30HX",
			sbOn:                  true,
			isWin11:               false,
			expectedModel:         GPUModelCMP30HX,
			expectedTensor:        false, // TU116 phần cứng không có Tensor Core
			expectedValorant:      true,
			expectedLMHT:          true,
			expectedNeedsSigning:  false,
			expectedRecommendedSB: "ENABLED",
			expectedBannerKeyword: "không cần nạp EFI Tensor Core",
		},
		{
			name:                  "30HX_Win10_SB_Off",
			deviceID:              DeviceIDCMP30HX,
			gpuName:               "NVIDIA CMP 30HX",
			sbOn:                  false,
			isWin11:               false,
			expectedModel:         GPUModelCMP30HX,
			expectedTensor:        false,
			expectedValorant:      true,
			expectedLMHT:          true,
			expectedNeedsSigning:  false,
			expectedRecommendedSB: "ENABLED",
			expectedBannerKeyword: "không cần nạp EFI Tensor Core",
		},
		{
			name:                  "30HX_Win11_SB_On_LyTuong",
			deviceID:              DeviceIDCMP30HX,
			gpuName:               "NVIDIA CMP 30HX",
			sbOn:                  true,
			isWin11:               true,
			expectedModel:         GPUModelCMP30HX,
			expectedTensor:        false,
			expectedValorant:      true,
			expectedLMHT:          true,
			expectedNeedsSigning:  false,
			expectedRecommendedSB: "ENABLED",
			expectedBannerKeyword: "không cần nạp EFI Tensor Core",
		},
		{
			name:                  "30HX_Win11_SB_Off_ValorantCanSB",
			deviceID:              DeviceIDCMP30HX,
			gpuName:               "NVIDIA CMP 30HX",
			sbOn:                  false,
			isWin11:               true,
			expectedModel:         GPUModelCMP30HX,
			expectedTensor:        false,
			expectedValorant:      false,
			expectedLMHT:          true,
			expectedNeedsSigning:  false,
			expectedRecommendedSB: "ENABLED",
			expectedBannerKeyword: "không cần nạp EFI Tensor Core",
		},
		{
			name:                  "Unknown_GPU",
			deviceID:              0x1000,
			gpuName:               "Unknown GPU",
			sbOn:                  true,
			isWin11:               true,
			expectedModel:         GPUModelUnknown,
			expectedTensor:        false,
			expectedValorant:      true,
			expectedLMHT:          true,
			expectedNeedsSigning:  false,
			expectedRecommendedSB: "ENABLED",
			expectedBannerKeyword: "Lưu ý: Hãy đảm bảo bạn đã mở khóa Gen 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			status := EvaluateRiotStatus(tt.deviceID, tt.gpuName, tt.sbOn, tt.isWin11)

			// Assert
			if status.Model != tt.expectedModel {
				t.Errorf("Model = %v; kỳ vọng %v", status.Model, tt.expectedModel)
			}
			if status.HasTensorCore != tt.expectedTensor {
				t.Errorf("HasTensorCore = %v; kỳ vọng %v", status.HasTensorCore, tt.expectedTensor)
			}
			if status.CanPlayValorant != tt.expectedValorant {
				t.Errorf("CanPlayValorant = %v; kỳ vọng %v", status.CanPlayValorant, tt.expectedValorant)
			}
			if status.CanPlayLeagueOfLegends != tt.expectedLMHT {
				t.Errorf("CanPlayLeagueOfLegends = %v; kỳ vọng %v", status.CanPlayLeagueOfLegends, tt.expectedLMHT)
			}
			if status.NeedsEFISigning != tt.expectedNeedsSigning {
				t.Errorf("NeedsEFISigning = %v; kỳ vọng %v", status.NeedsEFISigning, tt.expectedNeedsSigning)
			}
			if status.RecommendedSecureBoot != tt.expectedRecommendedSB {
				t.Errorf("RecommendedSecureBoot = %v; kỳ vọng %v", status.RecommendedSecureBoot, tt.expectedRecommendedSB)
			}
			if !strings.Contains(status.BannerText, tt.expectedBannerKeyword) {
				t.Errorf("BannerText không chứa keyword %q: %s", tt.expectedBannerKeyword, status.BannerText)
			}
			if status.CompletionTitle == "" {
				t.Errorf("CompletionTitle không được để trống")
			}
			if status.CompletionDetail == "" {
				t.Errorf("CompletionDetail không được để trống")
			}
		})
	}
}

func TestGetBiosRebootGuide(t *testing.T) {
	// Arrange
	resWithESP := &SigningResult{
		CertPathC:       `C:\CMP40HX_Key.cer`,
		CertPathDesktop: `C:\Users\Admin\Desktop\CMP40HX_Key.cer`,
		CertPathESP:     `S:\CMP40HX_Key.cer`,
	}

	resWithoutESP := &SigningResult{
		CertPathC:       `C:\CMP40HX_Key.cer`,
		CertPathDesktop: `C:\Users\Admin\Desktop\CMP40HX_Key.cer`,
	}

	t.Run("CoPhanVungESP", func(t *testing.T) {
		// Act
		guide := GetBiosRebootGuide(resWithESP)

		// Assert
		if !strings.Contains(guide, `S:\CMP40HX_Key.cer`) {
			t.Errorf("Guide phải chứa đường dẫn ESP")
		}
		if !strings.Contains(guide, "Key Management") {
			t.Errorf("Guide phải chứa hướng dẫn Key Management")
		}
		if !strings.Contains(guide, "Authorized Signatures (db)") {
			t.Errorf("Guide phải chứa hướng dẫn db enrollment")
		}
	})

	t.Run("KhongCoPhanVungESP", func(t *testing.T) {
		// Act
		guide := GetBiosRebootGuide(resWithoutESP)

		// Assert
		if strings.Contains(guide, "Phân vùng ESP:") {
			t.Errorf("Guide không được chứa dòng ESP nếu không có đường dẫn ESP")
		}
		if !strings.Contains(guide, `C:\CMP40HX_Key.cer`) {
			t.Errorf("Guide phải chứa đường dẫn ổ C")
		}
	})
}

func TestGetTensorGuide(t *testing.T) {
	tests := []struct {
		name           string
		isWin11        bool
		is40HX         bool
		expectKeywords []string
	}{
		{
			name:    "40HX_Win10",
			isWin11: false,
			is40HX:  true,
			expectKeywords: []string{
				"TẮT SECURE BOOT (Disabled)",
				"40HXUNLK.EFI",
				"SS0=0x88888888",
			},
		},
		{
			name:    "40HX_Win11",
			isWin11: true,
			is40HX:  true,
			expectKeywords: []string{
				"VALORANT (Bắt buộc bật Secure Boot + TPM)",
				"Append db",
				"Key Management (Custom Mode)",
				"Dual Boot",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			guide := GetTensorGuide(tt.isWin11, tt.is40HX)

			// Assert
			for _, kw := range tt.expectKeywords {
				if !strings.Contains(guide, kw) {
					t.Errorf("GetTensorGuide(%v, %v) không chứa từ khóa: %q", tt.isWin11, tt.is40HX, kw)
				}
			}
		})
	}
}

func TestRunOptimization_OutputLogging(t *testing.T) {
	tests := []struct {
		name           string
		is40HX         bool
		is30HX         bool
		sbOn           bool
		isWin11        bool
		expectedTokens []string
	}{
		{
			name:    "Log_40HX_Win10_SBOff",
			is40HX:  true,
			is30HX:  false,
			sbOn:    false,
			isWin11: false,
			expectedTokens: []string{
				"NVIDIA CMP 40HX [TU106]",
				"Trạng thái Secure Boot: ĐÃ TẮT",
				"BƯỚC 1: Dọn dẹp driver mở khóa",
				"BƯỚC 2: Tắt chế độ Windows Testsigning",
				"BƯỚC 3: Cấu hình Registry",
				"HOÀN TẤT TỐI ƯU HÓA",
			},
		},
		{
			name:    "Log_40HX_Win11_SBOn",
			is40HX:  true,
			is30HX:  false,
			sbOn:    true,
			isWin11: true,
			expectedTokens: []string{
				"NVIDIA CMP 40HX [TU106]",
				"Secure Boot hiện ĐANG BẬT",
				"BƯỚC 1: Dọn dẹp driver mở khóa",
				"BƯỚC 2: Tắt chế độ Windows Testsigning",
				"BƯỚC 3: Cấu hình Registry",
			},
		},
		{
			name:    "Log_30HX_Win11_SBOn",
			is40HX:  false,
			is30HX:  true,
			sbOn:    true,
			isWin11: true,
			expectedTokens: []string{
				"NVIDIA CMP 30HX [TU116]",
				"CMP 30HX không dùng EFI Tensor Core",
				"BƯỚC 1: Dọn dẹp driver mở khóa",
				"BƯỚC 2: Tắt chế độ Windows Testsigning",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			buf := &bytes.Buffer{}

			// Act
			runOptimization(buf, tt.is40HX, tt.is30HX, tt.sbOn, tt.isWin11)
			output := buf.String()

			// Assert
			for _, token := range tt.expectedTokens {
				if !strings.Contains(output, token) {
					t.Errorf("runOptimization output thiếu token: %q", token)
				}
			}
		})
	}
}
