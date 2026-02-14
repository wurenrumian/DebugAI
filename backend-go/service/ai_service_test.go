package service

import (
	"testing"

	"backend-go/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates a test database
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto migrate tables
	db.AutoMigrate(
		&models.User{},
		&models.AIRecord{},
		&models.WeakPoint{},
		&models.UserWeakPoint{},
	)

	return db
}

// TestAIService_ValidateEvaluateRequest tests the validation of evaluate requests
func TestValidateEvaluateRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *models.EvaluateRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &models.EvaluateRequest{
				StudentID:          "123456",
				Code:               "int main() {}",
				ProblemDescription: "Write a hello world program",
			},
			wantErr: false,
		},
		{
			name: "missing student_id",
			req: &models.EvaluateRequest{
				Code:               "int main() {}",
				ProblemDescription: "Write a hello world program",
			},
			wantErr: true,
		},
		{
			name: "missing code",
			req: &models.EvaluateRequest{
				StudentID:          "123456",
				ProblemDescription: "Write a hello world program",
			},
			wantErr: true,
		},
		{
			name: "missing problem description",
			req: &models.EvaluateRequest{
				StudentID: "123456",
				Code:      "int main() {}",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := models.ValidateEvaluateRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEvaluateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestAIService_ValidateRecommendRequest tests the validation of recommend requests
func TestValidateRecommendRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *models.RecommendRequest
		wantErr bool
	}{
		{
			name: "valid request with default max",
			req: &models.RecommendRequest{
				StudentID:          "123456",
				WeakPoints:         map[string]int{"循环": 3, "数组": 2},
				MaxRecommendations: 0, // Should be set to default 5
			},
			wantErr: false,
		},
		{
			name: "valid request with custom max",
			req: &models.RecommendRequest{
				StudentID:          "123456",
				WeakPoints:         map[string]int{"循环": 3},
				MaxRecommendations: 10,
			},
			wantErr: false,
		},
		{
			name: "missing student_id",
			req: &models.RecommendRequest{
				WeakPoints:         map[string]int{"循环": 3},
				MaxRecommendations: 5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := models.ValidateRecommendRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRecommendRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			// Check that max recommendations is set to default if 0
			if tt.req.MaxRecommendations == 0 && err == nil {
				if tt.req.MaxRecommendations != 5 {
					t.Errorf("MaxRecommendations should be set to 5, got %d", tt.req.MaxRecommendations)
				}
			}
		})
	}
}

// TestAIService_UpdateUserWeakPoints tests updating user weak points
func TestAIService_UpdateUserWeakPoints(t *testing.T) {
	db := setupTestDB(t)
	service := NewAIService(db, "http://localhost:8000")

	studentID := "test_student_123"
	weakPoints := map[string]int{
		"循环": 3,
		"数组": 2,
		"函数": 1,
	}

	// Update weak points
	err := service.UpdateUserWeakPoints(studentID, weakPoints)
	if err != nil {
		t.Fatalf("UpdateUserWeakPoints() error = %v", err)
	}

	// Verify weak points were created in dictionary
	var wpCount int64
	db.Model(&models.WeakPoint{}).Count(&wpCount)
	if wpCount != 3 {
		t.Errorf("Expected 3 weak points in dictionary, got %d", wpCount)
	}

	// Verify user weak points associations
	var userWPCount int64
	db.Model(&models.UserWeakPoint{}).Where("student_id = ?", studentID).Count(&userWPCount)
	if userWPCount != 3 {
		t.Errorf("Expected 3 user weak point associations, got %d", userWPCount)
	}
}

// TestAIService_GetTopWeakPoints tests getting top weak points
func TestAIService_GetTopWeakPoints(t *testing.T) {
	db := setupTestDB(t)
	service := NewAIService(db, "http://localhost:8000")

	studentID := "test_student_456"

	// Create weak points in the dictionary first
	wp1 := models.WeakPoint{Keyword: "循环", Category: "算法", Description: "循环相关"}
	wp2 := models.WeakPoint{Keyword: "数组", Category: "数据结构", Description: "数组相关"}
	wp3 := models.WeakPoint{Keyword: "函数", Category: "编程基础", Description: "函数相关"}
	db.Create(&wp1)
	db.Create(&wp2)
	db.Create(&wp3)

	// Create user weak points with different counts
	db.Create(&models.UserWeakPoint{StudentID: studentID, WeakPointID: wp1.ID, Count: 5})
	db.Create(&models.UserWeakPoint{StudentID: studentID, WeakPointID: wp2.ID, Count: 3})
	db.Create(&models.UserWeakPoint{StudentID: studentID, WeakPointID: wp3.ID, Count: 1})

	// Get top 2 weak points
	topPoints, err := service.GetTopWeakPoints(studentID, 2)
	if err != nil {
		t.Fatalf("GetTopWeakPoints() error = %v", err)
	}

	if len(topPoints) != 2 {
		t.Fatalf("Expected 2 top weak points, got %d", len(topPoints))
	}

	// First should be "循环" (count: 5)
	if topPoints[0] != "循环" {
		t.Errorf("Expected first weak point to be '循环', got '%s'", topPoints[0])
	}

	// Second should be "数组" (count: 3)
	if topPoints[1] != "数组" {
		t.Errorf("Expected second weak point to be '数组', got '%s'", topPoints[1])
	}
}

// TestAIService_SeedWeakPointKeywords tests seeding default keywords
func TestAIService_SeedWeakPointKeywords(t *testing.T) {
	db := setupTestDB(t)
	service := NewAIService(db, "http://localhost:8000")

	// Seed keywords
	err := service.SeedWeakPointKeywords()
	if err != nil {
		t.Fatalf("SeedWeakPointKeywords() error = %v", err)
	}

	// Verify keywords were created
	var count int64
	db.Model(&models.WeakPoint{}).Count(&count)
	if count == 0 {
		t.Error("Expected some weak points to be seeded, got 0")
	}

	// Seed again should not create duplicates
	err = service.SeedWeakPointKeywords()
	if err != nil {
		t.Fatalf("SeedWeakPointKeywords() second call error = %v", err)
	}

	var countAfterSecondSeed int64
	db.Model(&models.WeakPoint{}).Count(&countAfterSecondSeed)
	if count != countAfterSecondSeed {
		t.Errorf("Expected same count after second seed, before=%d, after=%d", count, countAfterSecondSeed)
	}
}

// TestAIService_GetUserWeakPoints tests getting user weak points
func TestAIService_GetUserWeakPoints(t *testing.T) {
	db := setupTestDB(t)
	service := NewAIService(db, "http://localhost:8000")

	studentID := "test_student_789"

	// Create weak points and user associations
	wp := models.WeakPoint{Keyword: "动态规划", Category: "算法", Description: "DP"}
	db.Create(&wp)
	db.Create(&models.UserWeakPoint{StudentID: studentID, WeakPointID: wp.ID, Count: 2})

	// Get user weak points
	weakPoints, err := service.GetUserWeakPoints(studentID)
	if err != nil {
		t.Fatalf("GetUserWeakPoints() error = %v", err)
	}

	if len(weakPoints) != 1 {
		t.Fatalf("Expected 1 weak point, got %d", len(weakPoints))
	}

	if weakPoints[0].Count != 2 {
		t.Errorf("Expected count to be 2, got %d", weakPoints[0].Count)
	}
}
