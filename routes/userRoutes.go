package routes

import (
	"backend/controllers"
	middleware "backend/middlewares"

	"github.com/gin-gonic/gin"
)

func UserRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.Use(middleware.Authenticate())
	incomingRoutes.GET("/admin/allAdmins", controllers.GetAllAdmins())
	incomingRoutes.GET("/admin/adminInfo", controllers.GetUserInfo())
	incomingRoutes.POST("/admin/sendRequest", controllers.SendRequest())
	incomingRoutes.PUT("/admin/modify-request/:id", controllers.ApproveOrRejectRequest())
	incomingRoutes.GET("/admin/all-requests", controllers.GetAllRequests())
	incomingRoutes.GET("/admin/adminsList", controllers.GetAllAdmins())
	incomingRoutes.GET("/admin/adminByID/:id", controllers.GetAdminByID())
	incomingRoutes.GET("/admin/client/all-clients", controllers.GetAllClients())
	incomingRoutes.GET("/admin/client/:id", controllers.GetClientByID())
	incomingRoutes.GET("/admin/jobListing/getJobListingById1/:id", controllers.GetJobListingById1())
}

func ClientRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.POST("/client/uploadPdf", controllers.UploadPdf())
	incomingRoutes.POST("/client/getPdfByEmail", controllers.GetPdfDetailsByUserEmail())
}

func JobListingRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.GET("/jobListings/getTest", controllers.GetTest())
	incomingRoutes.POST("/jobListings/createJobListing", controllers.CreateJobListing())
	incomingRoutes.GET("/jobListings/getAllJobListings", controllers.GetAllJobListings())
	incomingRoutes.GET("/jobListings/getJobListingById", controllers.GetJobListingById())
	incomingRoutes.POST("/jobListings/updateJobListing", controllers.UpdateJobListingByID())
	incomingRoutes.POST("/jobListings/deleteJobListing", controllers.DeleteJobListing())
}
