package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
	"github.com/stretchr/testify/require"
	mockdb "github.com/sangketkit01/simple-bank/db/mock"
	db "github.com/sangketkit01/simple-bank/db/sqlc"
	"github.com/sangketkit01/simple-bank/token"
	"github.com/sangketkit01/simple-bank/util"
)

// setupTest ฟังก์ชันสำหรับตั้งค่า mock objects ที่ใช้ในการทดสอบ
func setupTest(t *testing.T) (*Server, *mockdb.MockStore, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockStore := mockdb.NewMockStore(ctrl)
	config := util.Config{
		TokenSymmetricKey: util.RandomString(32),
		AccessTokenDuration: time.Minute,
	}
	server, err := NewServer(config, mockStore)
	require.NoError(t, err)
	return server, mockStore, ctrl
}

// createRandomAccount สร้างบัญชีแบบสุ่มสำหรับการทดสอบ
func createRandomAccount(owner string) db.Account {
	return db.Account{
		ID:       util.RandomInt(1, 1000),
		Owner:    owner,
		Balance:  util.RandomMoney(),
		Currency: util.RandomCurrency(),
		CreatedAt: time.Now(),
	}
}

// createRandomTransfer สร้างข้อมูล transfer แบบสุ่มสำหรับการทดสอบ
func createRandomTransfer(fromAccountID, toAccountID int64) db.Transfer {
	return db.Transfer{
		ID:            util.RandomInt(1, 1000),
		FromAccountID: sql.NullInt64{Int64: fromAccountID, Valid: true},
		ToAccountID:   sql.NullInt64{Int64: toAccountID, Valid: true},
		Amount:        util.RandomMoney(),
		CreatedAt:     time.Now(),
	}
}

// createRandomEntry สร้างข้อมูล entry แบบสุ่มสำหรับการทดสอบ
func createRandomEntry(accountID int64, amount int64) db.Entry {
	return db.Entry{
		ID:        util.RandomInt(1, 1000),
		AccountID: sql.NullInt64{Int64: accountID, Valid: true},
		Amount:    amount,
		CreatedAt: time.Now(),
	}
}

// TestCreateTransfer_Success ทดสอบการโอนเงินที่สำเร็จ
func TestCreateTransfer_Success(t *testing.T) {
	server, mockStore, ctrl := setupTest(t)
	defer ctrl.Finish()

	user1 := util.RandomOwner()
	user2 := util.RandomOwner()
	
	account1 := createRandomAccount(user1)
	account2 := createRandomAccount(user2)
	account1.Currency = "USD"
	account2.Currency = "USD"

	amount := int64(10)
	
	// สร้าง transfer result สำหรับการทดสอบ
	transferResult := db.TransferTxResult{
		Transfer: db.Transfer{
			ID:            util.RandomInt(1, 1000),
			FromAccountID: sql.NullInt64{Int64: account1.ID, Valid: true},
			ToAccountID:   sql.NullInt64{Int64: account2.ID, Valid: true},
			Amount:        amount,
			CreatedAt:     time.Now(),
		},
		FromAccount: account1,
		ToAccount:   account2,
		FromEntry: db.Entry{
			ID:        util.RandomInt(1, 1000),
			AccountID: sql.NullInt64{Int64: account1.ID, Valid: true},
			Amount:    -amount,
			CreatedAt: time.Now(),
		},
		ToEntry: db.Entry{
			ID:        util.RandomInt(1, 1000),
			AccountID: sql.NullInt64{Int64: account2.ID, Valid: true},
			Amount:    amount,
			CreatedAt: time.Now(),
		},
	}

	// กำหนดพฤติกรรมสำหรับ mock objects
	mockStore.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account1.ID)).Return(account1, nil)
	mockStore.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account2.ID)).Return(account2, nil)
	mockStore.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Return(transferResult, nil)

	// สร้าง token สำหรับการทดสอบ
	maker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)
	
	token,  err := maker.CreateToken(account1.Owner, time.Hour)
	require.NoError(t, err)

	// สร้าง request body
	requestBody := transferRequest{
		FromAccountID: account1.ID,
		ToAccountID:   account2.ID,
		Amount:        amount,
		Currency:      "USD",
	}
	bodyData, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// สร้าง HTTP request
	req, err := http.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(bodyData))
	require.NoError(t, err)
	
	// เพิ่ม authorization header
	authorizationHeader := fmt.Sprintf("Bearer %s", token)
	req.Header.Set("Authorization", authorizationHeader)
	req.Header.Set("Content-Type", "application/json")

	// บันทึก response
	recorder := httptest.NewRecorder()
	
	// ตั้งค่า gin router
	router := gin.Default()
	router.Use(authMiddleware(maker))
	router.POST("/transfers", server.createTransfer)
	
	// ทำการส่ง request
	router.ServeHTTP(recorder, req)
	
	// ตรวจสอบ response
	require.Equal(t, http.StatusOK, recorder.Code)
	
	var gotResponse db.TransferTxResult
	err = json.Unmarshal(recorder.Body.Bytes(), &gotResponse)
	require.NoError(t, err)
	require.Equal(t, transferResult.Transfer.FromAccountID, gotResponse.Transfer.FromAccountID)
	require.Equal(t, transferResult.Transfer.ToAccountID, gotResponse.Transfer.ToAccountID)
	require.Equal(t, transferResult.Transfer.Amount, gotResponse.Transfer.Amount)
}

// TestCreateTransfer_InvalidRequest ทดสอบกรณีข้อมูลที่ส่งมาไม่ถูกต้อง
func TestCreateTransfer_InvalidRequest(t *testing.T) {
	server, _, ctrl := setupTest(t)
	defer ctrl.Finish()

	// สร้าง token สำหรับการทดสอบ
	maker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)
	
	token, err := maker.CreateToken(util.RandomOwner(), time.Hour)
	require.NoError(t, err)

	testCases := []struct {
		name          string
		requestBody   gin.H
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "จำนวนเงินติดลบ",
			requestBody: gin.H{
				"from_account_id": 1,
				"to_account_id":   2,
				"amount":          -100,
				"currency":        "USD",
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "ไม่มี from_account_id",
			requestBody: gin.H{
				"to_account_id": 2,
				"amount":        100,
				"currency":      "USD",
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "ไม่มี to_account_id",
			requestBody: gin.H{
				"from_account_id": 1,
				"amount":          100,
				"currency":        "USD",
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "ไม่มีจำนวนเงิน",
			requestBody: gin.H{
				"from_account_id": 1,
				"to_account_id":   2,
				"currency":        "USD",
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "ไม่มี currency",
			requestBody: gin.H{
				"from_account_id": 1,
				"to_account_id":   2,
				"amount":          100,
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "currency ไม่ถูกต้อง",
			requestBody: gin.H{
				"from_account_id": 1,
				"to_account_id":   2,
				"amount":          100,
				"currency":        "INVALID",
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			// สร้าง request body
			bodyData, err := json.Marshal(tc.requestBody)
			require.NoError(t, err)

			// สร้าง HTTP request
			req, err := http.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(bodyData))
			require.NoError(t, err)
			
			// เพิ่ม authorization header
			authorizationHeader := fmt.Sprintf("Bearer %s", token)
			req.Header.Set("Authorization", authorizationHeader)
			req.Header.Set("Content-Type", "application/json")

			// บันทึก response
			recorder := httptest.NewRecorder()
			
			// ตั้งค่า gin router
			router := gin.Default()
			router.Use(authMiddleware(maker))
			router.POST("/transfers", server.createTransfer)
			
			// ทำการส่ง request
			router.ServeHTTP(recorder, req)
			
			// ตรวจสอบ response
			tc.checkResponse(recorder)
		})
	}
}

// TestCreateTransfer_FromAccountNotFound ทดสอบกรณีไม่พบบัญชีต้นทาง
func TestCreateTransfer_FromAccountNotFound(t *testing.T) {
	server, mockStore, ctrl := setupTest(t)
	defer ctrl.Finish()

	// กำหนดพฤติกรรมสำหรับ mock objects
	mockStore.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Return(db.Account{}, sql.ErrNoRows)

	// สร้าง token สำหรับการทดสอบ
	maker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)
	
	token,  err := maker.CreateToken(util.RandomOwner(), time.Hour)
	require.NoError(t, err)

	// สร้าง request body
	requestBody := transferRequest{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        100,
		Currency:      "USD",
	}
	bodyData, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// สร้าง HTTP request
	req, err := http.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(bodyData))
	require.NoError(t, err)
	
	// เพิ่ม authorization header
	authorizationHeader := fmt.Sprintf("Bearer %s", token)
	req.Header.Set("Authorization", authorizationHeader)
	req.Header.Set("Content-Type", "application/json")

	// บันทึก response
	recorder := httptest.NewRecorder()
	
	// ตั้งค่า gin router
	router := gin.Default()
	router.Use(authMiddleware(maker))
	router.POST("/transfers", server.createTransfer)
	
	// ทำการส่ง request
	router.ServeHTTP(recorder, req)
	
	// ตรวจสอบ response
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

// TestCreateTransfer_ToAccountNotFound ทดสอบกรณีไม่พบบัญชีปลายทาง
func TestCreateTransfer_ToAccountNotFound(t *testing.T) {
	server, mockStore, ctrl := setupTest(t)
	defer ctrl.Finish()
	
	user := util.RandomOwner()
	fromAccount := createRandomAccount(user)
	fromAccount.Currency = "USD"

	// กำหนดพฤติกรรมสำหรับ mock objects
	mockStore.EXPECT().GetAccount(gomock.Any(), gomock.Eq(fromAccount.ID)).Return(fromAccount, nil)
	mockStore.EXPECT().GetAccount(gomock.Any(), gomock.Not(fromAccount.ID)).Return(db.Account{}, sql.ErrNoRows)

	// สร้าง token สำหรับการทดสอบ
	maker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)
	
	token,  err := maker.CreateToken(fromAccount.Owner, time.Hour)
	require.NoError(t, err)

	// สร้าง request body
	requestBody := transferRequest{
		FromAccountID: fromAccount.ID,
		ToAccountID:   util.RandomInt(1, 1000),
		Amount:        100,
		Currency:      "USD",
	}
	bodyData, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// สร้าง HTTP request
	req, err := http.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(bodyData))
	require.NoError(t, err)
	
	// เพิ่ม authorization header
	authorizationHeader := fmt.Sprintf("Bearer %s", token)
	req.Header.Set("Authorization", authorizationHeader)
	req.Header.Set("Content-Type", "application/json")

	// บันทึก response
	recorder := httptest.NewRecorder()
	
	// ตั้งค่า gin router
	router := gin.Default()
	router.Use(authMiddleware(maker))
	router.POST("/transfers", server.createTransfer)
	
	// ทำการส่ง request
	router.ServeHTTP(recorder, req)
	
	// ตรวจสอบ response
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

// TestCreateTransfer_FromAccountCurrencyMismatch ทดสอบกรณีสกุลเงินของบัญชีต้นทางไม่ตรงกับที่ระบุ
func TestCreateTransfer_FromAccountCurrencyMismatch(t *testing.T) {
	server, mockStore, ctrl := setupTest(t)
	defer ctrl.Finish()
	
	user := util.RandomOwner()
	fromAccount := createRandomAccount(user)
	fromAccount.Currency = "USD"

	// กำหนดพฤติกรรมสำหรับ mock objects
	mockStore.EXPECT().GetAccount(gomock.Any(), gomock.Eq(fromAccount.ID)).Return(fromAccount, nil)

	// สร้าง token สำหรับการทดสอบ
	maker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)
	
	token, err := maker.CreateToken(fromAccount.Owner, time.Hour)
	require.NoError(t, err)

	// สร้าง request body ที่มี currency ไม่ตรงกับบัญชี
	requestBody := transferRequest{
		FromAccountID: fromAccount.ID,
		ToAccountID:   util.RandomInt(1, 1000),
		Amount:        100,
		Currency:      "EUR", // ไม่ตรงกับ USD ที่เป็นสกุลเงินของบัญชี
	}
	bodyData, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// สร้าง HTTP request
	req, err := http.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(bodyData))
	require.NoError(t, err)
	
	// เพิ่ม authorization header
	authorizationHeader := fmt.Sprintf("Bearer %s", token)
	req.Header.Set("Authorization", authorizationHeader)
	req.Header.Set("Content-Type", "application/json")

	// บันทึก response
	recorder := httptest.NewRecorder()
	
	// ตั้งค่า gin router
	router := gin.Default()
	router.Use(authMiddleware(maker))
	router.POST("/transfers", server.createTransfer)
	
	// ทำการส่ง request
	router.ServeHTTP(recorder, req)
	
	// ตรวจสอบ response
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

// TestCreateTransfer_ToAccountCurrencyMismatch ทดสอบกรณีสกุลเงินของบัญชีปลายทางไม่ตรงกับที่ระบุ
func TestCreateTransfer_ToAccountCurrencyMismatch(t *testing.T) {
	server, mockStore, ctrl := setupTest(t)
	defer ctrl.Finish()
	
	user := util.RandomOwner()
	fromAccount := createRandomAccount(user)
	fromAccount.Currency = "USD"
	
	toAccount := createRandomAccount(util.RandomOwner())
	toAccount.Currency = "EUR" // สกุลเงินต่างจากที่จะระบุในคำขอ

	// กำหนดพฤติกรรมสำหรับ mock objects
	mockStore.EXPECT().GetAccount(gomock.Any(), gomock.Eq(fromAccount.ID)).Return(fromAccount, nil)
	mockStore.EXPECT().GetAccount(gomock.Any(), gomock.Eq(toAccount.ID)).Return(toAccount, nil)

	// สร้าง token สำหรับการทดสอบ
	maker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)
	
	token,  err := maker.CreateToken(fromAccount.Owner, time.Hour)
	require.NoError(t, err)

	// สร้าง request body
	requestBody := transferRequest{
		FromAccountID: fromAccount.ID,
		ToAccountID:   toAccount.ID,
		Amount:        100,
		Currency:      "USD", // ตรงกับบัญชีต้นทาง แต่ไม่ตรงกับบัญชีปลายทาง
	}
	bodyData, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// สร้าง HTTP request
	req, err := http.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(bodyData))
	require.NoError(t, err)
	
	// เพิ่ม authorization header
	authorizationHeader := fmt.Sprintf("Bearer %s", token)
	req.Header.Set("Authorization", authorizationHeader)
	req.Header.Set("Content-Type", "application/json")

	// บันทึก response
	recorder := httptest.NewRecorder()
	
	// ตั้งค่า gin router
	router := gin.Default()
	router.Use(authMiddleware(maker))
	router.POST("/transfers", server.createTransfer)
	
	// ทำการส่ง request
	router.ServeHTTP(recorder, req)
	
	// ตรวจสอบ response
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

// TestCreateTransfer_UnauthorizedUser ทดสอบกรณีผู้ใช้ไม่มีสิทธิ์โอนเงินจากบัญชี
func TestCreateTransfer_UnauthorizedUser(t *testing.T) {
	server, mockStore, ctrl := setupTest(t)
	defer ctrl.Finish()
	
	fromAccount := createRandomAccount(util.RandomOwner())
	fromAccount.Currency = "USD"

	// กำหนดพฤติกรรมสำหรับ mock objects
	mockStore.EXPECT().GetAccount(gomock.Any(), gomock.Eq(fromAccount.ID)).Return(fromAccount, nil)

	// สร้าง token สำหรับการทดสอบด้วยชื่อผู้ใช้ที่ไม่ตรงกับเจ้าของบัญชี
	maker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)
	
	differentUser := util.RandomOwner() // สร้างชื่อผู้ใช้ที่ไม่ใช่เจ้าของบัญชี
	token,  err := maker.CreateToken(differentUser, time.Hour)
	require.NoError(t, err)

	// สร้าง request body
	requestBody := transferRequest{
		FromAccountID: fromAccount.ID,
		ToAccountID:   util.RandomInt(1, 1000),
		Amount:        100,
		Currency:      "USD",
	}
	bodyData, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// สร้าง HTTP request
	req, err := http.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(bodyData))
	require.NoError(t, err)
	
	// เพิ่ม authorization header
	authorizationHeader := fmt.Sprintf("Bearer %s", token)
	req.Header.Set("Authorization", authorizationHeader)
	req.Header.Set("Content-Type", "application/json")

	// บันทึก response
	recorder := httptest.NewRecorder()
	
	// ตั้งค่า gin router
	router := gin.Default()
	router.Use(authMiddleware(maker))
	router.POST("/transfers", server.createTransfer)
	
	// ทำการส่ง request
	router.ServeHTTP(recorder, req)
	
	// ตรวจสอบ response
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

// TestCreateTransfer_DBError ทดสอบกรณีเกิดข้อผิดพลาดในการทำธุรกรรม
func TestCreateTransfer_DBError(t *testing.T) {
	server, mockStore, ctrl := setupTest(t)
	defer ctrl.Finish()
	
	user := util.RandomOwner()
	fromAccount := createRandomAccount(user)
	fromAccount.Currency = "USD"
	
	toAccount := createRandomAccount(util.RandomOwner())
	toAccount.Currency = "USD"

	// กำหนดพฤติกรรมสำหรับ mock objects
	mockStore.EXPECT().GetAccount(gomock.Any(), gomock.Eq(fromAccount.ID)).Return(fromAccount, nil)
	mockStore.EXPECT().GetAccount(gomock.Any(), gomock.Eq(toAccount.ID)).Return(toAccount, nil)
	mockStore.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Return(db.TransferTxResult{}, sql.ErrConnDone)

	// สร้าง token สำหรับการทดสอบ
	maker, err := token.NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)
	
	token, err := maker.CreateToken(fromAccount.Owner, time.Hour)
	require.NoError(t, err)

	// สร้าง request body
	requestBody := transferRequest{
		FromAccountID: fromAccount.ID,
		ToAccountID:   toAccount.ID,
		Amount:        100,
		Currency:      "USD",
	}
	bodyData, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// สร้าง HTTP request
	req, err := http.NewRequest(http.MethodPost, "/transfers", bytes.NewReader(bodyData))
	require.NoError(t, err)
	
	// เพิ่ม authorization header
	authorizationHeader := fmt.Sprintf("Bearer %s", token)
	req.Header.Set("Authorization", authorizationHeader)
	req.Header.Set("Content-Type", "application/json")

	// บันทึก response
	recorder := httptest.NewRecorder()
	
	// ตั้งค่า gin router
	router := gin.Default()
	router.Use(authMiddleware(maker))
	router.POST("/transfers", server.createTransfer)
	
	// ทำการส่ง request
	router.ServeHTTP(recorder, req)
	
	// ตรวจสอบ response
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}
