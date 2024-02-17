package controllers

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"backend/database"
	helper "backend/helpers"
	"backend/models"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var clientCollection *mongo.Collection = database.OpenCollection(database.Client, "clients")
var pdfUploadsCollection *mongo.Collection = database.OpenCollection(database.Client, "pdfUploads")
var validateClient = validator.New()

func HashPasswordClient(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		log.Panic(err)
	}
	return string(bytes)
}

func VerifyPasswordClient(userPassword string, providedPassword string) (bool, string) {
	err := bcrypt.CompareHashAndPassword([]byte(providedPassword), []byte(userPassword))
	check := err == nil
	msg := ""
	if !check {
		msg = fmt.Sprintf("email or password not matched")
	}
	return check, msg
}

func ClientRegister() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var client models.Client

		if err := c.BindJSON(&client); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		validationErr := validateClient.Struct(client)

		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": validationErr.Error(),
			})
			return
		}

		countByEmail, err := clientCollection.CountDocuments(ctx, bson.M{"email": client.Email})
		if err != nil {
			log.Panic(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "error occurred",
			})
			return
		}

		countByContact, err := clientCollection.CountDocuments(ctx, bson.M{"contact": client.Contact})
		if err != nil {
			log.Panic(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "error occurred",
			})
			return
		}

		// Check if either email or contact already exists
		if countByEmail > 0 && countByContact > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "this email or contact already exists",
			})
			return
		}

		password := HashPasswordClient(*client.Password)
		client.Password = &password

		client.ID = primitive.NewObjectID()
		clientID := client.ID.Hex()
		client.User_id = &clientID

		if client.Email == nil || client.Name == nil || client.Company == nil || client.Password == nil || client.Contact == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "missing required fields",
			})
			return
		}

		token, refreshToken := helper.GenerateAllTokens(
			*client.Email,
			*client.Name,
			*client.Company,
			*client.User_id,
			*client.Contact,
		)
		client.Token = &token
		client.Refresh_token = &refreshToken

		_, insertErr := clientCollection.InsertOne(ctx, client)
		if insertErr != nil {
			msg := fmt.Sprintf("client item was not created")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": msg,
			})
			return
		}

		defer cancel()
		c.JSON(http.StatusOK, gin.H{
			"message": "client registered successfully",
			"client":  client,
		})

		return
	}
}

func ClientLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var client models.Client
		var foundClient models.Client

		if err := c.BindJSON(&client); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		err := clientCollection.FindOne(ctx, bson.M{"email": client.Email}).Decode(&foundClient)
		defer cancel()

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "email or password not matched",
			})
			return
		}

		passwordIsValid, msg := VerifyPasswordClient(*client.Password, *foundClient.Password)
		if !passwordIsValid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": msg,
			})
			return
		}

		if foundClient.Email == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "HELLO",
			})
			return
		}

		token, refreshToken := helper.GenerateAllTokens(
			*foundClient.Email,
			*foundClient.Name,
			*foundClient.Company,
			*foundClient.User_id,
			*foundClient.Contact,
		)
		helper.UpdateAllTokens(token, refreshToken, *foundClient.User_id)
		err = clientCollection.FindOne(ctx, bson.M{"user_id": *foundClient.User_id}).Decode(&foundClient)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		defer cancel()
		c.JSON(http.StatusOK, gin.H{
			"message": "Login successful",
			"client":  foundClient,
		})
	}
}

func SendRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		var request models.Request_to_admin

		// Parse the incoming JSON request body into the Requests struct
		if err := c.BindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		// Get the current time and format it as desired
		currentTime := time.Now().Format("2006-01-02 15:04:05")
		request.Sended_At = &currentTime

		// Insert the request data into the MongoDB collection
		result, err := requestCollection.InsertOne(context.Background(), request)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		// Respond with success message and inserted data
		c.JSON(http.StatusOK, gin.H{
			"msg":    "successfully request sent",
			"data":   request,
			"status": result,
		})
	}
}

func UploadPdf() gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("UploadPdf handler function called")

		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		// Parse form
		fmt.Println("Parsing form...")
		err := c.Request.ParseMultipartForm(10 << 20) // 10 MB max
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			fmt.Println("Error parsing form:", err.Error())
			return
		}

		// Get the PDF file from the request
		fmt.Println("Getting PDF file from request...")
		file, _, err := c.Request.FormFile("pdf_file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "PDF file not found in request",
			})
			fmt.Println("PDF file not found in request")
			return
		}
		defer file.Close()

		// Read the file content
		fmt.Println("Reading PDF file content...")
		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to read PDF file",
			})
			fmt.Println("Failed to read PDF file:", err.Error())
			return
		}

		// Get other fields from the request
		title := c.Request.FormValue("title")
		userEmail := c.Request.FormValue("user_email")

		// Generate a new unique ObjectID
		id := primitive.NewObjectID()

		// Create PDFUploads struct
		pdfUpload := models.PDFUploads{
			ID:        id,
			Title:     title,
			UserEmail: userEmail,
			PDFFile:   fileBytes,
		}

		// Validate PDFUploads struct
		fmt.Println("Validating PDFUploads struct...")
		validationErr := validateClient.Struct(pdfUpload)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": validationErr.Error(),
			})
			fmt.Println("Validation error:", validationErr.Error())
			return
		}

		// Insert PDFUploads data into the MongoDB collection
		fmt.Println("Inserting PDFUploads data into MongoDB collection...")
		result, err := pdfUploadsCollection.InsertOne(ctx, pdfUpload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			fmt.Println("Error inserting data into MongoDB collection:", err.Error())
			return
		}

		// Respond with success message and inserted data
		fmt.Println("PDF uploaded successfully")
		c.JSON(http.StatusOK, gin.H{
			"msg":    "PDF uploaded successfully",
			"data":   pdfUpload,
			"status": result,
		})
	}
}

func GetPdfDetailsByUserEmail() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse the JSON request body
		var requestBody map[string]string
		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		// Retrieve the user email from the request body
		userEmail, ok := requestBody["useremail"]
		if !ok || userEmail == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User email is required"})
			return
		}

		// Retrieve the PDF details from the database based on the user email
		pdfDetails, err := GetPdfDetailsByUserEmailFromDatabase(userEmail)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve PDF details"})
			return
		}

		// Set response headers
		c.Header("Content-Type", "application/json")

		// Send the PDF details as the response
		c.JSON(http.StatusOK, pdfDetails)
	}
}

func GetPdfDetailsByUserEmailFromDatabase(userEmail string) ([]bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := pdfUploadsCollection.Find(ctx, bson.M{"useremail": userEmail})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var pdfDetails []bson.M
	err = cursor.All(ctx, &pdfDetails)
	if err != nil {
		return nil, err
	}

	return pdfDetails, nil
}
