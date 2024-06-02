package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"

	"net/http"

	"github.com/alexanderthegreat96/mongo-db-api-go/driver"
	"github.com/alexanderthegreat96/mongo-db-api-go/helpers"
	"github.com/alexanderthegreat96/mongo-db-api-go/responses"
	"github.com/common-nighthawk/go-figure"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var apiPort string
var apiHost string

func init() {
	// ensure loading .env file in the
	// root of the probam
	// otherwise driver cannot acccess
	// it's contents

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	logger := log.New(os.Stdout, "[MONGO-API]: ", log.Ldate|log.Ltime)

	apiPort = helpers.GetEnv("API_PORT", "9776")
	apiHost = helpers.GetEnv("API_HOST", "localhost")

	versionNumber := "v1.0"
	mongoApiBanner := figure.NewColorFigure("MongoAPI "+versionNumber, "", "blue", false)
	mongoApiBanner.Print()

	logger.Println("Server Information:")
	logger.Printf("MongoDB Host: %s", helpers.GetEnv("MONGO_DB_HOST", ""))
	logger.Printf("MongoDB Port: %s", helpers.GetEnv("MONGO_DB_PORT", ""))
	logger.Printf("MongoDB Username: %s", helpers.GetEnv("MONGO_DB_USERNAME", "admin"))
	logger.Printf("MongoDB Password: %s", helpers.GetEnv("MONGO_DB_HOST", "admin"))
	logger.Printf("MongoDB Default Database: %s", helpers.GetEnv("MONGO_DB_NAME", ""))
	logger.Printf("MongoDB Default Table: %s", helpers.GetEnv("MONGO_DB_Table", ""))

	logger.Println("API Information:")
	logger.Println("You may start sending requests to: http://" + apiHost + ":" + apiPort)
	logger.Println("Please open up your browser and head over to: /docs")

}

func main() {
	// remove when NOT compiling
	gin.SetMode(gin.ReleaseMode)
	mongoApi := gin.Default()

	// list all databses on the server
	mongoApi.GET("/db/databases", func(c *gin.Context) {
		databases, err := driver.MongoDB().ListDatabases()

		if err.Error != "" {
			c.JSON(http.StatusOK, responses.GenericErrorResponse{
				Code:   err.Code,
				Status: err.Status,
				Error:  err.Error,
			})
		} else {
			c.JSON(http.StatusOK, responses.DatabaseListResponse{
				Status:    true,
				Databases: databases.Databases,
			})
		}
	})

	// Delete a database
	mongoApi.DELETE("/db/:db_name/delete", func(c *gin.Context) {
		dbName := c.Param("db_name")
		if dbName != "" {
			unableToWipeDbErr := driver.MongoDB().DropDatabase(dbName)
			if unableToWipeDbErr.Error != "" {
				c.JSON(http.StatusOK, responses.GenericErrorResponse{
					Code:   unableToWipeDbErr.Code,
					Status: unableToWipeDbErr.Status,
					Error:  unableToWipeDbErr.Error,
				})
			} else {
				c.JSON(http.StatusOK, responses.DeleteDatabaseSuccessResponse{
					Status:  true,
					Message: "Database: " + dbName + " has been deleted!",
				})
			}

		} else {
			c.JSON(http.StatusBadRequest, responses.GenericErrorResponse{
				Code:   400,
				Status: false,
				Error:  "Failed to provide a database name",
			})
		}

	})

	// list all tables / collections in a database
	mongoApi.GET("/db/:db_name/tables", func(c *gin.Context) {
		dbName := c.Param("db_name")
		if dbName != "" {
			tablesInDatabase, tablesInDatabaseErr := driver.MongoDB().ListCollections(dbName)
			if tablesInDatabaseErr.Error != "" {
				c.JSON(http.StatusOK, responses.GenericErrorResponse{
					Code:   tablesInDatabaseErr.Code,
					Status: tablesInDatabaseErr.Status,
					Error:  tablesInDatabaseErr.Error,
				})
			} else {
				c.JSON(http.StatusOK, responses.TablesInDatabaseResponse{
					Status: tablesInDatabase.Status,
					Tables: tablesInDatabase.Tables,
				})
			}
		} else {
			c.JSON(http.StatusBadRequest, responses.GenericErrorResponse{
				Code:   400,
				Status: false,
				Error:  "Failed to provide a database name",
			})
		}

	})

	// delete a collection inside a database
	mongoApi.DELETE("/db/:db_name/:table_name/delete", func(c *gin.Context) {
		dbName := c.Param("db_name")
		tableName := c.Param("table_name")
		if dbName != "" && tableName != "" {
			wipeTableErr := driver.MongoDB().DropTable(dbName, tableName)
			if wipeTableErr.Error != "" {
				c.JSON(http.StatusOK, responses.GenericErrorResponse{
					Code:   wipeTableErr.Code,
					Status: wipeTableErr.Status,
					Error:  wipeTableErr.Error,
				})
			} else {
				c.JSON(http.StatusOK, responses.WipeTableInDatabaseResponse{
					Status:  true,
					Message: "Table / collection: " + tableName + ", dropped!",
				})
			}
		} else {
			c.JSON(http.StatusBadRequest, responses.GenericErrorResponse{
				Code:   400,
				Status: false,
				Error:  "Failed to provide a database name / table name.",
			})
		}

	})

	// retrieve data from a database > collection

	mongoApi.GET("/db/:db_name/:table_name/select", func(c *gin.Context) {
		dbName := c.Param("db_name")
		tableName := c.Param("table_name")

		page := 1
		if c.Query("page") != "" {
			pageNumber, pageNumberErr := strconv.Atoi(c.Query("page"))
			if pageNumberErr != nil {
				log.Printf("Issue converting page number to int: %s", pageNumberErr.Error())
			} else {
				page = pageNumber
			}
		}

		perPage := 10
		if c.Query("per_page") != "" {
			perPageNumber, perPageNumberErr := strconv.Atoi(c.Query("per_page"))
			if perPageNumberErr != nil {
				log.Printf("Issue converting page number to int: %s", perPageNumberErr.Error())
			} else {
				perPage = perPageNumber
			}
		}

		var sort [][]interface{}

		if c.Query("sort") != "" {
			sort = helpers.ParseSort(c.Query("sort"))
		}

		var andQuery [][]interface{}

		if c.Query("query_and") != "" {
			andQuery = helpers.ParseQuery(c.Query("query_and"))
		}

		var orQuery [][]interface{}

		if c.Query("query_or") != "" {
			orQuery = helpers.ParseQuery(c.Query("query_or"))
		}

		results, resultsErr := driver.
			MongoDB().
			DB(dbName).
			Table(tableName).
			Page(page).
			PerPage(perPage).
			AndAll(andQuery).
			OrAll(orQuery).
			SortAll(sort).
			Find()

		var rawQuery interface{}
		if resultsErr.Query != "" {
			rawQuery = json.RawMessage(resultsErr.Query)
		}
		if results.Query != "" {
			rawQuery = json.RawMessage(results.Query)
		}

		if resultsErr.Error != "" {
			c.JSON(http.StatusOK, responses.GenericErrorResponse{
				Code:   resultsErr.Code,
				Status: resultsErr.Status,
				Error:  resultsErr.Error,
				Query:  rawQuery,
			})
		} else {
			c.JSON(http.StatusOK, responses.SelectResultsResponse{
				Status:   results.Status,
				Code:     results.Code,
				Database: results.Database,
				Table:    results.Table,
				Count:    results.Count,
				Pagination: responses.SelectResultsPaginationResponse{
					TotalPages:  results.Pagination.TotalPages,
					CurrentPage: results.Pagination.CurrentPage,
					NextPage:    results.Pagination.NextPage,
					PrevPage:    results.Pagination.PrevPage,
					LastPage:    results.Pagination.LastPage,
					PerPage:     results.Pagination.PerPage,
				},
				Query:   rawQuery,
				Results: results.Results,
			})
		}
	})
	mongoApi.Run(apiHost + ":" + apiPort) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
