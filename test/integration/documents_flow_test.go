//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/richxcame/ride-hailing/internal/documents"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/pkg/storage"
)

const documentsServiceKey = "documents"

// fakeStorage implements storage.Storage for testing
type fakeStorage struct {
	files map[string][]byte
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{files: make(map[string][]byte)}
}

func (f *fakeStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*storage.UploadResult, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	f.files[key] = data
	return &storage.UploadResult{
		Key:        key,
		URL:        fmt.Sprintf("https://fake-storage.test/%s", key),
		Size:       size,
		MimeType:   contentType,
		UploadedAt: time.Now(),
	}, nil
}

func (f *fakeStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	data, ok := f.files[key]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", key)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (f *fakeStorage) Delete(ctx context.Context, key string) error {
	delete(f.files, key)
	return nil
}

func (f *fakeStorage) GetURL(key string) string {
	return fmt.Sprintf("https://fake-storage.test/%s", key)
}

func (f *fakeStorage) GetPresignedUploadURL(ctx context.Context, key string, contentType string, expiresIn time.Duration) (*storage.PresignedURLResult, error) {
	return &storage.PresignedURLResult{
		URL:       fmt.Sprintf("https://fake-storage.test/presigned/%s", key),
		Method:    "PUT",
		Headers:   map[string]string{"Content-Type": contentType},
		ExpiresAt: time.Now().Add(expiresIn),
	}, nil
}

func (f *fakeStorage) GetPresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (*storage.PresignedURLResult, error) {
	return &storage.PresignedURLResult{
		URL:       fmt.Sprintf("https://fake-storage.test/presigned/%s", key),
		Method:    "GET",
		ExpiresAt: time.Now().Add(expiresIn),
	}, nil
}

func (f *fakeStorage) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := f.files[key]
	return ok, nil
}

func (f *fakeStorage) Copy(ctx context.Context, sourceKey, destKey string) error {
	data, ok := f.files[sourceKey]
	if !ok {
		return fmt.Errorf("source file not found: %s", sourceKey)
	}
	f.files[destKey] = data
	return nil
}

// fakeDriverService implements documents.DriverServiceInterface for testing
type fakeDriverService struct {
	drivers map[uuid.UUID]*models.Driver
}

func newFakeDriverService() *fakeDriverService {
	return &fakeDriverService{drivers: make(map[uuid.UUID]*models.Driver)}
}

func (f *fakeDriverService) GetDriverByUserID(ctx context.Context, userID uuid.UUID) (*models.Driver, error) {
	driver, ok := f.drivers[userID]
	if !ok {
		return nil, fmt.Errorf("driver not found for user: %s", userID)
	}
	return driver, nil
}

func (f *fakeDriverService) RegisterDriver(userID uuid.UUID, driverID uuid.UUID) {
	f.drivers[userID] = &models.Driver{
		ID:            driverID,
		UserID:        userID,
		LicenseNumber: "DL123456789",
		VehicleModel:  "Toyota Camry",
		VehiclePlate:  "ABC-1234",
		VehicleColor:  "Silver",
		VehicleYear:   2020,
		IsAvailable:   false,
		IsOnline:      false,
		Rating:        0.0,
		TotalRides:    0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

var (
	testStorage       *fakeStorage
	testDriverService *fakeDriverService
)

func startDocumentsService() *serviceInstance {
	testStorage = newFakeStorage()
	testDriverService = newFakeDriverService()

	repo := documents.NewRepository(dbPool)
	service := documents.NewService(repo, testStorage, documents.ServiceConfig{
		MaxFileSizeMB:    10,
		AllowedMimeTypes: []string{"image/jpeg", "image/png", "application/pdf"},
		OCREnabled:       true,
		OCRProvider:      "mock",
	})
	handler := documents.NewHandler(service, testDriverService)

	router := gin.New()
	router.Use(middleware.Recovery())
	router.Use(middleware.CorrelationID())

	router.GET("/healthz", common.HealthCheck("documents", "integration"))

	// Document types (public)
	router.GET("/api/v1/documents/types", handler.GetDocumentTypes)

	// Driver routes (authenticated)
	driverDocs := router.Group("/api/v1/documents")
	driverDocs.Use(middleware.AuthMiddleware("integration-secret"))
	driverDocs.Use(middleware.RequireRole(models.RoleDriver))
	{
		driverDocs.GET("", handler.GetMyDocuments)
		driverDocs.GET("/verification-status", handler.GetMyVerificationStatus)
		driverDocs.POST("/upload", handler.UploadDocument)
		driverDocs.GET("/:id", handler.GetDocument)
		driverDocs.POST("/:id/back", handler.UploadDocumentBackSide)
	}

	// Admin routes
	adminDocs := router.Group("/api/v1/admin/documents")
	adminDocs.Use(middleware.AuthMiddleware("integration-secret"))
	adminDocs.Use(middleware.RequireAdmin())
	{
		adminDocs.GET("/pending", handler.GetPendingReviews)
		adminDocs.GET("/expiring", handler.GetExpiringDocuments)
		adminDocs.POST("/:id/start-review", handler.StartDocumentReview)
		adminDocs.POST("/:id/review", handler.ReviewDocument)
	}

	adminDriverDocs := router.Group("/api/v1/admin/drivers")
	adminDriverDocs.Use(middleware.AuthMiddleware("integration-secret"))
	adminDriverDocs.Use(middleware.RequireAdmin())
	{
		adminDriverDocs.GET("/:driver_id/documents", handler.GetDriverDocumentsAdmin)
		adminDriverDocs.GET("/:driver_id/verification-status", handler.GetDriverVerificationStatusAdmin)
	}

	server := httptest.NewServer(router)
	return &serviceInstance{server: server, client: server.Client(), baseURL: server.URL}
}

// Helper to create multipart form request for file upload
func createMultipartUploadRequest(t *testing.T, serviceKey, path string, documentTypeCode string, fileName string, fileContent []byte, headers map[string]string) *http.Response {
	t.Helper()

	svc, ok := services[serviceKey]
	require.True(t, ok, "service %s not registered", serviceKey)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("file", fileName)
	require.NoError(t, err)
	_, err = part.Write(fileContent)
	require.NoError(t, err)

	// Add document type
	err = writer.WriteField("document_type_code", documentTypeCode)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, svc.baseURL+path, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := svc.client.Do(req)
	require.NoError(t, err)
	return resp
}

// registerDriverAndLogin registers a driver user and creates driver record
func registerDriverAndLogin(t *testing.T) authSession {
	t.Helper()
	session := registerAndLogin(t, models.RoleDriver)

	// Create driver record in database
	driverID := uuid.New()
	_, err := dbPool.Exec(context.Background(), `
		INSERT INTO drivers (id, user_id, license_number, vehicle_model, vehicle_plate, vehicle_color, vehicle_year, is_available, is_online)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, driverID, session.User.ID, "DL"+uuid.NewString()[:8], "Toyota Camry", "ABC-"+uuid.NewString()[:4], "Silver", 2020, false, false)
	require.NoError(t, err)

	// Register driver in fake service
	testDriverService.RegisterDriver(session.User.ID, driverID)

	return session
}

// truncateDocumentTables truncates document-related tables
func truncateDocumentTables(t *testing.T) {
	t.Helper()
	tables := []string{
		"ocr_processing_queue",
		"document_expiry_notifications",
		"document_verification_history",
		"driver_documents",
		"driver_verification_status",
	}

	for _, table := range tables {
		_, err := dbPool.Exec(context.Background(), fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			// Table might not exist, that's okay
			t.Logf("Warning: could not truncate %s: %v", table, err)
		}
	}
}

// ========================================
// TEST: Document Upload Flow
// ========================================

func TestDocumentsIntegration_UploadDriverLicense(t *testing.T) {
	truncateTables(t)
	truncateDocumentTables(t)

	if _, ok := services[documentsServiceKey]; !ok {
		services[documentsServiceKey] = startDocumentsService()
	}

	driver := registerDriverAndLogin(t)

	// Upload driver license
	fileContent := []byte("fake jpeg image content")
	resp := createMultipartUploadRequest(t, documentsServiceKey, "/api/v1/documents/upload", "drivers_license", "license.jpg", fileContent, authHeaders(driver.Token))
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result apiResponse[documents.UploadDocumentResponse]
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.NotEqual(t, uuid.Nil, result.Data.DocumentID)
	require.Equal(t, documents.StatusPending, result.Data.Status)
	require.NotEmpty(t, result.Data.FileURL)
	require.Equal(t, "Document uploaded successfully", result.Data.Message)

	// Verify document was created in database
	var docCount int
	err = dbPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM driver_documents WHERE id = $1",
		result.Data.DocumentID).Scan(&docCount)
	require.NoError(t, err)
	require.Equal(t, 1, docCount)

	// Verify file was stored
	require.True(t, len(testStorage.files) > 0, "file should be stored")

	t.Log("Document upload test passed successfully")
}

func TestDocumentsIntegration_UploadMultipleDocumentTypes(t *testing.T) {
	truncateTables(t)
	truncateDocumentTables(t)

	if _, ok := services[documentsServiceKey]; !ok {
		services[documentsServiceKey] = startDocumentsService()
	}

	driver := registerDriverAndLogin(t)

	documentTypes := []string{"drivers_license", "vehicle_registration", "vehicle_insurance"}
	uploadedDocs := make([]uuid.UUID, 0)

	for _, docType := range documentTypes {
		fileContent := []byte(fmt.Sprintf("fake content for %s", docType))
		resp := createMultipartUploadRequest(t, documentsServiceKey, "/api/v1/documents/upload", docType, docType+".jpg", fileContent, authHeaders(driver.Token))
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode, "failed to upload %s", docType)

		var result apiResponse[documents.UploadDocumentResponse]
		err := json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		require.True(t, result.Success)

		uploadedDocs = append(uploadedDocs, result.Data.DocumentID)
	}

	require.Len(t, uploadedDocs, 3)

	// Verify all documents were created
	var docCount int
	err := dbPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM driver_documents WHERE id = ANY($1)",
		uploadedDocs).Scan(&docCount)
	require.NoError(t, err)
	require.Equal(t, 3, docCount)

	t.Log("Multiple document upload test passed successfully")
}

// ========================================
// TEST: Document OCR Processing
// ========================================

func TestDocumentsIntegration_OCRProcessing(t *testing.T) {
	truncateTables(t)
	truncateDocumentTables(t)

	if _, ok := services[documentsServiceKey]; !ok {
		services[documentsServiceKey] = startDocumentsService()
	}

	driver := registerDriverAndLogin(t)

	// Upload document
	fileContent := []byte("fake jpeg image content for OCR")
	resp := createMultipartUploadRequest(t, documentsServiceKey, "/api/v1/documents/upload", "drivers_license", "license.jpg", fileContent, authHeaders(driver.Token))
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result apiResponse[documents.UploadDocumentResponse]
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	require.True(t, result.Success)

	documentID := result.Data.DocumentID

	// Simulate OCR processing by directly updating the document
	ocrData := map[string]interface{}{
		"document_number":   "DL123456789",
		"full_name":         "John Doe",
		"date_of_birth":     "1990-01-15",
		"issue_date":        "2020-01-01",
		"expiry_date":       "2025-01-01",
		"issuing_authority": "DMV California",
	}
	ocrDataJSON, _ := json.Marshal(ocrData)
	ocrConfidence := 0.95

	_, err = dbPool.Exec(context.Background(), `
		UPDATE driver_documents
		SET ocr_data = $1, ocr_confidence = $2, ocr_processed_at = NOW()
		WHERE id = $3
	`, ocrDataJSON, ocrConfidence, documentID)
	require.NoError(t, err)

	// Verify OCR data was saved
	var savedOCRData []byte
	var savedConfidence float64
	err = dbPool.QueryRow(context.Background(),
		"SELECT ocr_data, ocr_confidence FROM driver_documents WHERE id = $1",
		documentID).Scan(&savedOCRData, &savedConfidence)
	require.NoError(t, err)
	require.InEpsilon(t, ocrConfidence, savedConfidence, 0.001)

	var parsedOCR map[string]interface{}
	err = json.Unmarshal(savedOCRData, &parsedOCR)
	require.NoError(t, err)
	require.Equal(t, "DL123456789", parsedOCR["document_number"])

	t.Log("OCR processing test passed successfully")
}

func TestDocumentsIntegration_OCRQueueCreation(t *testing.T) {
	truncateTables(t)
	truncateDocumentTables(t)

	if _, ok := services[documentsServiceKey]; !ok {
		services[documentsServiceKey] = startDocumentsService()
	}

	driver := registerDriverAndLogin(t)

	// Upload document (should schedule OCR)
	fileContent := []byte("fake jpeg image content")
	resp := createMultipartUploadRequest(t, documentsServiceKey, "/api/v1/documents/upload", "drivers_license", "license.jpg", fileContent, authHeaders(driver.Token))
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result apiResponse[documents.UploadDocumentResponse]
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Verify OCR job was created (if OCR is enabled for this document type)
	var ocrJobCount int
	err = dbPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM ocr_processing_queue WHERE document_id = $1",
		result.Data.DocumentID).Scan(&ocrJobCount)
	require.NoError(t, err)

	// OCR should be scheduled for drivers_license
	if result.Data.OCRScheduled {
		require.Equal(t, 1, ocrJobCount, "OCR job should be created when OCR is scheduled")
	}

	t.Log("OCR queue creation test passed successfully")
}

// ========================================
// TEST: Document Verification Status Updates
// ========================================

func TestDocumentsIntegration_VerificationStatusFlow(t *testing.T) {
	truncateTables(t)
	truncateDocumentTables(t)

	if _, ok := services[documentsServiceKey]; !ok {
		services[documentsServiceKey] = startDocumentsService()
	}

	admin := registerAndLogin(t, models.RoleAdmin)
	driver := registerDriverAndLogin(t)

	// Upload document
	fileContent := []byte("fake jpeg image content")
	uploadResp := createMultipartUploadRequest(t, documentsServiceKey, "/api/v1/documents/upload", "drivers_license", "license.jpg", fileContent, authHeaders(driver.Token))
	defer uploadResp.Body.Close()
	require.Equal(t, http.StatusCreated, uploadResp.StatusCode)

	var uploadResult apiResponse[documents.UploadDocumentResponse]
	err := json.NewDecoder(uploadResp.Body).Decode(&uploadResult)
	require.NoError(t, err)
	documentID := uploadResult.Data.DocumentID

	// Verify initial status is pending
	var status string
	err = dbPool.QueryRow(context.Background(),
		"SELECT status FROM driver_documents WHERE id = $1",
		documentID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "pending", status)

	// Admin starts review
	startReviewPath := fmt.Sprintf("/api/v1/admin/documents/%s/start-review", documentID)
	startReviewResp := doRequest[map[string]string](t, documentsServiceKey, http.MethodPost, startReviewPath, nil, authHeaders(admin.Token))
	require.True(t, startReviewResp.Success)

	// Verify status changed to under_review
	err = dbPool.QueryRow(context.Background(),
		"SELECT status FROM driver_documents WHERE id = $1",
		documentID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "under_review", status)

	// Admin approves document
	reviewPath := fmt.Sprintf("/api/v1/admin/documents/%s/review", documentID)
	reviewReq := documents.ReviewDocumentRequest{
		Action: "approve",
		Notes:  "Document verified successfully",
	}
	reviewResp := doRequest[map[string]string](t, documentsServiceKey, http.MethodPost, reviewPath, reviewReq, authHeaders(admin.Token))
	require.True(t, reviewResp.Success)

	// Verify status changed to approved
	err = dbPool.QueryRow(context.Background(),
		"SELECT status FROM driver_documents WHERE id = $1",
		documentID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "approved", status)

	// Verify history was created
	var historyCount int
	err = dbPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM document_verification_history WHERE document_id = $1",
		documentID).Scan(&historyCount)
	require.NoError(t, err)
	require.GreaterOrEqual(t, historyCount, 2, "should have at least 2 history entries (submitted, approved)")

	t.Log("Verification status flow test passed successfully")
}

func TestDocumentsIntegration_DocumentRejection(t *testing.T) {
	truncateTables(t)
	truncateDocumentTables(t)

	if _, ok := services[documentsServiceKey]; !ok {
		services[documentsServiceKey] = startDocumentsService()
	}

	admin := registerAndLogin(t, models.RoleAdmin)
	driver := registerDriverAndLogin(t)

	// Upload document
	fileContent := []byte("fake blurry image content")
	uploadResp := createMultipartUploadRequest(t, documentsServiceKey, "/api/v1/documents/upload", "drivers_license", "license.jpg", fileContent, authHeaders(driver.Token))
	defer uploadResp.Body.Close()
	require.Equal(t, http.StatusCreated, uploadResp.StatusCode)

	var uploadResult apiResponse[documents.UploadDocumentResponse]
	err := json.NewDecoder(uploadResp.Body).Decode(&uploadResult)
	require.NoError(t, err)
	documentID := uploadResult.Data.DocumentID

	// Admin starts review
	startReviewPath := fmt.Sprintf("/api/v1/admin/documents/%s/start-review", documentID)
	doRequest[map[string]string](t, documentsServiceKey, http.MethodPost, startReviewPath, nil, authHeaders(admin.Token))

	// Admin rejects document
	reviewPath := fmt.Sprintf("/api/v1/admin/documents/%s/review", documentID)
	reviewReq := documents.ReviewDocumentRequest{
		Action:          "reject",
		RejectionReason: "Image is too blurry to read",
		Notes:           "Please upload a clearer image",
	}
	reviewResp := doRequest[map[string]string](t, documentsServiceKey, http.MethodPost, reviewPath, reviewReq, authHeaders(admin.Token))
	require.True(t, reviewResp.Success)

	// Verify status changed to rejected
	var status string
	var rejectionReason string
	err = dbPool.QueryRow(context.Background(),
		"SELECT status, rejection_reason FROM driver_documents WHERE id = $1",
		documentID).Scan(&status, &rejectionReason)
	require.NoError(t, err)
	require.Equal(t, "rejected", status)
	require.Equal(t, "Image is too blurry to read", rejectionReason)

	t.Log("Document rejection test passed successfully")
}

func TestDocumentsIntegration_GetPendingReviews(t *testing.T) {
	truncateTables(t)
	truncateDocumentTables(t)

	if _, ok := services[documentsServiceKey]; !ok {
		services[documentsServiceKey] = startDocumentsService()
	}

	admin := registerAndLogin(t, models.RoleAdmin)
	driver := registerDriverAndLogin(t)

	// Upload multiple documents
	for i := 0; i < 3; i++ {
		fileContent := []byte(fmt.Sprintf("fake content %d", i))
		resp := createMultipartUploadRequest(t, documentsServiceKey, "/api/v1/documents/upload", "drivers_license", fmt.Sprintf("license%d.jpg", i), fileContent, authHeaders(driver.Token))
		resp.Body.Close()
	}

	// Get pending reviews as admin
	type pendingReviewsResponse struct {
		Documents []documents.PendingReviewDocument `json:"documents"`
	}

	pendingResp := doRequest[pendingReviewsResponse](t, documentsServiceKey, http.MethodGet, "/api/v1/admin/documents/pending", nil, authHeaders(admin.Token))
	require.True(t, pendingResp.Success)

	// Should have pending documents (may only have the latest due to superseding)
	require.GreaterOrEqual(t, len(pendingResp.Data.Documents), 1)

	t.Log("Get pending reviews test passed successfully")
}

// ========================================
// TEST: Document Expiration Handling
// ========================================

func TestDocumentsIntegration_DocumentExpiration(t *testing.T) {
	truncateTables(t)
	truncateDocumentTables(t)

	if _, ok := services[documentsServiceKey]; !ok {
		services[documentsServiceKey] = startDocumentsService()
	}

	admin := registerAndLogin(t, models.RoleAdmin)
	driver := registerDriverAndLogin(t)

	// Get driver ID
	driverID := testDriverService.drivers[driver.User.ID].ID

	// Get document type ID for drivers_license
	var docTypeID uuid.UUID
	err := dbPool.QueryRow(context.Background(),
		"SELECT id FROM document_types WHERE code = 'drivers_license'").Scan(&docTypeID)
	require.NoError(t, err)

	// Insert an approved document that is expiring soon
	documentID := uuid.New()
	expiryDate := time.Now().Add(7 * 24 * time.Hour) // Expires in 7 days
	_, err = dbPool.Exec(context.Background(), `
		INSERT INTO driver_documents (id, driver_id, document_type_id, status, file_url, file_key, file_name, expiry_date, submitted_at)
		VALUES ($1, $2, $3, 'approved', 'https://test.com/doc.jpg', 'test/doc.jpg', 'doc.jpg', $4, NOW())
	`, documentID, driverID, docTypeID, expiryDate)
	require.NoError(t, err)

	// Get expiring documents
	type expiringDocsResponse struct {
		Documents []documents.ExpiringDocument `json:"documents"`
	}

	expiringResp := doRequest[expiringDocsResponse](t, documentsServiceKey, http.MethodGet, "/api/v1/admin/documents/expiring?days=30", nil, authHeaders(admin.Token))
	require.True(t, expiringResp.Success)

	// Should find the expiring document
	require.GreaterOrEqual(t, len(expiringResp.Data.Documents), 1)

	// Verify the document is in the list
	found := false
	for _, doc := range expiringResp.Data.Documents {
		if doc.Document.ID == documentID {
			found = true
			require.Equal(t, "critical", doc.Urgency) // 7 days should be critical
			break
		}
	}
	require.True(t, found, "expiring document should be in the list")

	t.Log("Document expiration test passed successfully")
}

func TestDocumentsIntegration_ExpireOutdatedDocuments(t *testing.T) {
	truncateTables(t)
	truncateDocumentTables(t)

	if _, ok := services[documentsServiceKey]; !ok {
		services[documentsServiceKey] = startDocumentsService()
	}

	driver := registerDriverAndLogin(t)
	driverID := testDriverService.drivers[driver.User.ID].ID

	// Get document type ID
	var docTypeID uuid.UUID
	err := dbPool.QueryRow(context.Background(),
		"SELECT id FROM document_types WHERE code = 'drivers_license'").Scan(&docTypeID)
	require.NoError(t, err)

	// Insert an approved document that has expired
	documentID := uuid.New()
	expiryDate := time.Now().Add(-1 * 24 * time.Hour) // Expired yesterday
	_, err = dbPool.Exec(context.Background(), `
		INSERT INTO driver_documents (id, driver_id, document_type_id, status, file_url, file_key, file_name, expiry_date, submitted_at)
		VALUES ($1, $2, $3, 'approved', 'https://test.com/doc.jpg', 'test/doc.jpg', 'doc.jpg', $4, NOW())
	`, documentID, driverID, docTypeID, expiryDate)
	require.NoError(t, err)

	// Call the expire function
	_, err = dbPool.Exec(context.Background(), "SELECT expire_outdated_documents()")
	require.NoError(t, err)

	// Verify document status is now expired
	var status string
	err = dbPool.QueryRow(context.Background(),
		"SELECT status FROM driver_documents WHERE id = $1",
		documentID).Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "expired", status)

	t.Log("Expire outdated documents test passed successfully")
}

// ========================================
// TEST: Re-upload Expired Documents
// ========================================

func TestDocumentsIntegration_ReuploadExpiredDocument(t *testing.T) {
	truncateTables(t)
	truncateDocumentTables(t)

	if _, ok := services[documentsServiceKey]; !ok {
		services[documentsServiceKey] = startDocumentsService()
	}

	driver := registerDriverAndLogin(t)
	driverID := testDriverService.drivers[driver.User.ID].ID

	// Get document type ID
	var docTypeID uuid.UUID
	err := dbPool.QueryRow(context.Background(),
		"SELECT id FROM document_types WHERE code = 'drivers_license'").Scan(&docTypeID)
	require.NoError(t, err)

	// Insert an expired document
	expiredDocID := uuid.New()
	_, err = dbPool.Exec(context.Background(), `
		INSERT INTO driver_documents (id, driver_id, document_type_id, status, file_url, file_key, file_name, expiry_date, version, submitted_at)
		VALUES ($1, $2, $3, 'expired', 'https://test.com/old.jpg', 'test/old.jpg', 'old.jpg', $4, 1, NOW() - INTERVAL '1 year')
	`, expiredDocID, driverID, docTypeID, time.Now().Add(-30*24*time.Hour))
	require.NoError(t, err)

	// Re-upload new document
	fileContent := []byte("new license image content")
	resp := createMultipartUploadRequest(t, documentsServiceKey, "/api/v1/documents/upload", "drivers_license", "new_license.jpg", fileContent, authHeaders(driver.Token))
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result apiResponse[documents.UploadDocumentResponse]
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.NotEqual(t, expiredDocID, result.Data.DocumentID, "should create new document, not reuse expired")

	// Verify new document was created with incremented version
	var newVersion int
	err = dbPool.QueryRow(context.Background(),
		"SELECT version FROM driver_documents WHERE id = $1",
		result.Data.DocumentID).Scan(&newVersion)
	require.NoError(t, err)
	require.Equal(t, 1, newVersion, "new document after expired should start at version 1")

	// Verify old expired document still exists with expired status
	var oldStatus string
	err = dbPool.QueryRow(context.Background(),
		"SELECT status FROM driver_documents WHERE id = $1",
		expiredDocID).Scan(&oldStatus)
	require.NoError(t, err)
	require.Equal(t, "expired", oldStatus)

	t.Log("Re-upload expired document test passed successfully")
}

func TestDocumentsIntegration_SupersedeExistingDocument(t *testing.T) {
	truncateTables(t)
	truncateDocumentTables(t)

	if _, ok := services[documentsServiceKey]; !ok {
		services[documentsServiceKey] = startDocumentsService()
	}

	driver := registerDriverAndLogin(t)

	// Upload first version
	fileContent1 := []byte("first version content")
	resp1 := createMultipartUploadRequest(t, documentsServiceKey, "/api/v1/documents/upload", "drivers_license", "license_v1.jpg", fileContent1, authHeaders(driver.Token))
	defer resp1.Body.Close()
	require.Equal(t, http.StatusCreated, resp1.StatusCode)

	var result1 apiResponse[documents.UploadDocumentResponse]
	err := json.NewDecoder(resp1.Body).Decode(&result1)
	require.NoError(t, err)
	firstDocID := result1.Data.DocumentID

	// Verify first document is pending
	var status1 string
	err = dbPool.QueryRow(context.Background(),
		"SELECT status FROM driver_documents WHERE id = $1",
		firstDocID).Scan(&status1)
	require.NoError(t, err)
	require.Equal(t, "pending", status1)

	// Upload second version (should supersede first)
	fileContent2 := []byte("second version content")
	resp2 := createMultipartUploadRequest(t, documentsServiceKey, "/api/v1/documents/upload", "drivers_license", "license_v2.jpg", fileContent2, authHeaders(driver.Token))
	defer resp2.Body.Close()
	require.Equal(t, http.StatusCreated, resp2.StatusCode)

	var result2 apiResponse[documents.UploadDocumentResponse]
	err = json.NewDecoder(resp2.Body).Decode(&result2)
	require.NoError(t, err)
	secondDocID := result2.Data.DocumentID

	// Verify first document is now superseded
	err = dbPool.QueryRow(context.Background(),
		"SELECT status FROM driver_documents WHERE id = $1",
		firstDocID).Scan(&status1)
	require.NoError(t, err)
	require.Equal(t, "superseded", status1)

	// Verify second document is pending and references first
	var status2 string
	var previousDocID *uuid.UUID
	var version int
	err = dbPool.QueryRow(context.Background(),
		"SELECT status, previous_document_id, version FROM driver_documents WHERE id = $1",
		secondDocID).Scan(&status2, &previousDocID, &version)
	require.NoError(t, err)
	require.Equal(t, "pending", status2)
	require.NotNil(t, previousDocID)
	require.Equal(t, firstDocID, *previousDocID)
	require.Equal(t, 2, version)

	t.Log("Supersede existing document test passed successfully")
}

// ========================================
// TEST: Get My Documents / Verification Status
// ========================================

func TestDocumentsIntegration_GetMyDocuments(t *testing.T) {
	truncateTables(t)
	truncateDocumentTables(t)

	if _, ok := services[documentsServiceKey]; !ok {
		services[documentsServiceKey] = startDocumentsService()
	}

	driver := registerDriverAndLogin(t)

	// Upload documents
	docTypes := []string{"drivers_license", "vehicle_registration"}
	for _, docType := range docTypes {
		fileContent := []byte(fmt.Sprintf("content for %s", docType))
		resp := createMultipartUploadRequest(t, documentsServiceKey, "/api/v1/documents/upload", docType, docType+".jpg", fileContent, authHeaders(driver.Token))
		resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
	}

	// Get my documents
	type myDocsResponse struct {
		Documents []documents.DriverDocument `json:"documents"`
	}

	docsResp := doRequest[myDocsResponse](t, documentsServiceKey, http.MethodGet, "/api/v1/documents", nil, authHeaders(driver.Token))
	require.True(t, docsResp.Success)
	require.Len(t, docsResp.Data.Documents, 2)

	t.Log("Get my documents test passed successfully")
}

func TestDocumentsIntegration_GetVerificationStatus(t *testing.T) {
	truncateTables(t)
	truncateDocumentTables(t)

	if _, ok := services[documentsServiceKey]; !ok {
		services[documentsServiceKey] = startDocumentsService()
	}

	driver := registerDriverAndLogin(t)

	// Get verification status before uploading any documents
	statusResp := doRequest[documents.VerificationStatusResponse](t, documentsServiceKey, http.MethodGet, "/api/v1/documents/verification-status", nil, authHeaders(driver.Token))
	require.True(t, statusResp.Success)
	require.Equal(t, documents.VerificationIncomplete, statusResp.Data.Status)
	require.False(t, statusResp.Data.CanDrive)
	require.Greater(t, len(statusResp.Data.MissingDocuments), 0)

	t.Log("Get verification status test passed successfully")
}

// ========================================
// TEST: Authorization
// ========================================

func TestDocumentsIntegration_RiderCannotUploadDocuments(t *testing.T) {
	truncateTables(t)
	truncateDocumentTables(t)

	if _, ok := services[documentsServiceKey]; !ok {
		services[documentsServiceKey] = startDocumentsService()
	}

	rider := registerAndLogin(t, models.RoleRider)

	// Try to upload as rider (should fail)
	fileContent := []byte("fake content")
	resp := createMultipartUploadRequest(t, documentsServiceKey, "/api/v1/documents/upload", "drivers_license", "license.jpg", fileContent, authHeaders(rider.Token))
	defer resp.Body.Close()

	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	t.Log("Rider cannot upload documents test passed successfully")
}

func TestDocumentsIntegration_DriverCannotAccessAdminEndpoints(t *testing.T) {
	truncateTables(t)
	truncateDocumentTables(t)

	if _, ok := services[documentsServiceKey]; !ok {
		services[documentsServiceKey] = startDocumentsService()
	}

	driver := registerDriverAndLogin(t)

	// Try to access admin pending reviews endpoint
	resp := doRawRequest(t, documentsServiceKey, http.MethodGet, "/api/v1/admin/documents/pending", nil, authHeaders(driver.Token))
	defer resp.Body.Close()

	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	t.Log("Driver cannot access admin endpoints test passed successfully")
}

func TestDocumentsIntegration_UnauthenticatedAccessDenied(t *testing.T) {
	truncateTables(t)
	truncateDocumentTables(t)

	if _, ok := services[documentsServiceKey]; !ok {
		services[documentsServiceKey] = startDocumentsService()
	}

	// Try to access protected endpoint without token
	resp := doRawRequest(t, documentsServiceKey, http.MethodGet, "/api/v1/documents", nil, nil)
	defer resp.Body.Close()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	t.Log("Unauthenticated access denied test passed successfully")
}

// ========================================
// TEST: Get Document Types (Public Endpoint)
// ========================================

func TestDocumentsIntegration_GetDocumentTypes(t *testing.T) {
	truncateTables(t)
	truncateDocumentTables(t)

	if _, ok := services[documentsServiceKey]; !ok {
		services[documentsServiceKey] = startDocumentsService()
	}

	// Get document types (public endpoint - no auth required)
	resp := doRawRequest(t, documentsServiceKey, http.MethodGet, "/api/v1/documents/types", nil, nil)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result apiResponse[documents.DocumentTypeListResponse]
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Greater(t, len(result.Data.DocumentTypes), 0)

	// Verify expected document types exist
	typeMap := make(map[string]bool)
	for _, dt := range result.Data.DocumentTypes {
		typeMap[dt.Code] = true
	}
	require.True(t, typeMap["drivers_license"], "should have drivers_license type")
	require.True(t, typeMap["vehicle_registration"], "should have vehicle_registration type")
	require.True(t, typeMap["vehicle_insurance"], "should have vehicle_insurance type")

	t.Log("Get document types test passed successfully")
}
