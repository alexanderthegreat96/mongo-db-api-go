package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/alexanderthegreat96/mongo-db-api-go/driver"
	"github.com/alexanderthegreat96/mongo-db-api-go/helpers"
	"github.com/alexanderthegreat96/mongo-db-api-go/responses"
	"github.com/gin-gonic/gin"
)

func RunApi(mongoDb driver.MongoDBHandler, apiHost string, apiPort string) {
	// remove when NOT compiling
	// gin.SetMode(gin.ReleaseMode)
	// @BasePath /
	mongoApi := gin.Default()

	// list all databses on the server
	mongoApi.GET("/db/databases", func(c *gin.Context) {
		databases, databasesErr := mongoDb.ListDatabases()

		if databasesErr.Error != "" {
			c.JSON(databasesErr.Code, responses.GenericErrorResponse{
				Code:   databasesErr.Code,
				Status: databasesErr.Status,
				Error:  databasesErr.Error,
			})
		} else {
			c.JSON(databases.Code, responses.DatabaseListResponse{
				Status:    true,
				Databases: databases.Databases,
			})
		}
	})

	// Delete a database
	mongoApi.DELETE("/db/:db_name/delete", func(c *gin.Context) {
		dbName := c.Param("db_name")
		unableToWipeDbErr := mongoDb.DropDatabase(dbName)
		if unableToWipeDbErr.Error != "" {
			c.JSON(unableToWipeDbErr.Code, responses.GenericErrorResponse{
				Code:     unableToWipeDbErr.Code,
				Status:   unableToWipeDbErr.Status,
				Database: unableToWipeDbErr.Database,
				Error:    unableToWipeDbErr.Error,
			})
		} else {
			c.JSON(http.StatusOK, responses.DeleteDatabaseSuccessResponse{
				Status:  true,
				Message: "Database: " + dbName + " has been deleted!",
			})
		}

	})

	// list all tables / collections in a database
	mongoApi.GET("/db/:db_name/tables", func(c *gin.Context) {
		dbName := c.Param("db_name")
		tablesInDatabase, tablesInDatabaseErr := mongoDb.ListCollections(dbName)
		if tablesInDatabaseErr.Error != "" {
			c.JSON(tablesInDatabaseErr.Code, responses.GenericErrorResponse{
				Code:     tablesInDatabaseErr.Code,
				Status:   tablesInDatabaseErr.Status,
				Database: tablesInDatabaseErr.Database,
				Error:    tablesInDatabaseErr.Error,
			})
		} else {
			c.JSON(tablesInDatabase.Code, responses.TablesInDatabaseResponse{
				Status: tablesInDatabase.Status,
				Tables: tablesInDatabase.Tables,
			})
		}

	})

	// delete a collection inside a database
	mongoApi.DELETE("/db/:db_name/:table_name/delete", func(c *gin.Context) {
		dbName := c.Param("db_name")
		tableName := c.Param("table_name")

		wipeTableErr := mongoDb.DropTable(dbName, tableName)
		if wipeTableErr.Error != "" {
			c.JSON(wipeTableErr.Code, responses.GenericErrorResponse{
				Code:     wipeTableErr.Code,
				Status:   wipeTableErr.Status,
				Database: wipeTableErr.Database,
				Table:    wipeTableErr.Table,
				Error:    wipeTableErr.Error,
			})
		} else {
			c.JSON(http.StatusOK, responses.WipeTableInDatabaseResponse{
				Status:  true,
				Message: "Table / collection: " + tableName + ", dropped!",
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

		mongoDb.ResetQuery()
		mongoDb.ResetSort()

		results, resultsErr :=
			mongoDb.
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
			c.JSON(results.Code, responses.GenericErrorResponse{
				Code:     resultsErr.Code,
				Status:   resultsErr.Status,
				Error:    resultsErr.Error,
				Database: resultsErr.Database,
				Table:    resultsErr.Table,
				Query:    rawQuery,
			})
			return
		}

		c.JSON(results.Code, responses.SelectResultsResponse{
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
	})

	// retrieve a record by ID
	mongoApi.GET("/db/:db_name/:table_name/get/:mongo_id", func(c *gin.Context) {
		dbName := c.Param("db_name")
		tableName := c.Param("table_name")
		mongoId := c.Param("mongo_id")

		findOne, findOneErr := mongoDb.DB(dbName).Table(tableName).FindById(mongoId)

		if findOneErr.Error != "" {
			c.JSON(findOneErr.Code, responses.GenericErrorResponse{
				Code:     findOneErr.Code,
				Status:   findOneErr.Status,
				Error:    findOneErr.Error,
				Database: findOneErr.Database,
				Table:    findOneErr.Table,
			})
			return
		}

		c.JSON(findOne.Code, responses.SelectSingleResultResponse{
			Status:   findOne.Status,
			Code:     findOne.Code,
			Database: findOne.Database,
			Table:    findOne.Table,
			Result:   findOne.Result,
		})
	})

	// insert data into the database
	mongoApi.POST("/db/:db_name/:table_name/insert", func(c *gin.Context) {
		dbName := c.Param("db_name")
		tableName := c.Param("table_name")

		err := c.Request.ParseForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		payload := c.Request.Form.Get("payload")
		if payload == "" {
			c.JSON(http.StatusBadRequest, responses.GenericErrorResponse{
				Code:     400,
				Status:   false,
				Error:    "Failed to provide data under the key 'payload'",
				Database: dbName,
				Table:    tableName,
			})
			return
		}

		convertedPayload, convertedPayloadErr := helpers.ConvertJsonToData(payload)
		if convertedPayloadErr != nil {
			c.JSON(http.StatusBadRequest, responses.GenericErrorResponse{
				Code:     400,
				Status:   false,
				Error:    convertedPayloadErr.Error(),
				Database: dbName,
				Table:    tableName,
			})
			return
		}

		insert, insertErr := mongoDb.DB(dbName).Table(tableName).Insert(convertedPayload)
		if insertErr.Error != "" {
			c.JSON(http.StatusOK, responses.GenericErrorResponse{
				Code:     insertErr.Code,
				Status:   insertErr.Status,
				Error:    insertErr.Error,
				Database: insertErr.Database,
				Table:    insertErr.Table,
			})
			return
		}

		c.JSON(http.StatusOK, responses.MongoOperationsResultResponse{
			Code:      insert.Code,
			Status:    insert.Status,
			Database:  insert.Database,
			Table:     insert.Table,
			Operation: insert.Operation,
			Message:   insert.Message,
		})

	})

	// update data by id
	mongoApi.PUT("/db/:db_name/:table_name/update/:mongo_id", func(c *gin.Context) {
		dbName := c.Param("db_name")
		tableName := c.Param("table_name")
		mongoId := c.Param("mongo_id")

		err := c.Request.ParseForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		payload := c.Request.Form.Get("payload")
		if payload == "" {
			c.JSON(http.StatusBadRequest, responses.GenericErrorResponse{
				Code:     400,
				Status:   false,
				Error:    "Failed to provide data under the key 'payload'",
				Database: dbName,
				Table:    tableName,
			})
			return
		}

		convertedPayload, convertedPayloadErr := helpers.ConvertJsonToData(payload)
		if convertedPayloadErr != nil {
			c.JSON(http.StatusBadRequest, responses.GenericErrorResponse{
				Code:     400,
				Status:   false,
				Error:    convertedPayloadErr.Error(),
				Database: dbName,
				Table:    tableName,
			})
			return
		}

		updateById, updateByIdErr := mongoDb.DB(dbName).Table(tableName).UpdateByID(mongoId, convertedPayload)

		if updateByIdErr.Error != "" {
			c.JSON(updateByIdErr.Code, responses.GenericErrorResponse{
				Code:     updateByIdErr.Code,
				Status:   updateByIdErr.Status,
				Error:    updateByIdErr.Error,
				Database: updateByIdErr.Database,
				Table:    updateByIdErr.Table,
			})
			return
		}

		c.JSON(updateById.Code, responses.MongoOperationsResultResponse{
			Code:      updateById.Code,
			Status:    updateById.Status,
			Database:  updateById.Database,
			Table:     updateById.Table,
			Operation: updateById.Operation,
			Message:   updateById.Message,
		})
	})

	// update based on a provided query
	mongoApi.PUT("/db/:db_name/:table_name/update-where", func(c *gin.Context) {
		dbName := c.Param("db_name")
		tableName := c.Param("table_name")

		err := c.Request.ParseForm()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		payload := c.Request.Form.Get("payload")
		if payload == "" {
			c.JSON(http.StatusBadRequest, responses.GenericErrorResponse{
				Code:     400,
				Status:   false,
				Error:    "Failed to provide data under the key 'payload'",
				Database: dbName,
				Table:    tableName,
			})
			return
		}

		convertedPayload, convertedPayloadErr := helpers.ConvertJsonToData(payload)
		if convertedPayloadErr != nil {
			c.JSON(http.StatusBadRequest, responses.GenericErrorResponse{
				Code:     400,
				Status:   false,
				Error:    convertedPayloadErr.Error(),
				Database: dbName,
				Table:    tableName,
			})
			return
		}

		var andQuery [][]interface{}

		if c.Query("query_and") != "" {
			andQuery = helpers.ParseQuery(c.Query("query_and"))
		}

		var orQuery [][]interface{}

		if c.Query("query_or") != "" {
			orQuery = helpers.ParseQuery(c.Query("query_or"))
		}

		mongoDb.ResetQuery()
		update, updateErr :=
			mongoDb.
				DB(dbName).
				Table(tableName).
				AndAll(andQuery).
				OrAll(orQuery).
				Update(convertedPayload)

		var rawQuery interface{}
		if updateErr.Query != "" {
			rawQuery = json.RawMessage(updateErr.Query)
		}
		if update.Query != "" {
			rawQuery = json.RawMessage(update.Query)
		}

		if updateErr.Error != "" {
			c.JSON(updateErr.Code, responses.GenericErrorResponse{
				Code:     updateErr.Code,
				Status:   updateErr.Status,
				Error:    updateErr.Error,
				Database: updateErr.Database,
				Table:    updateErr.Table,
				Query:    rawQuery,
			})
			return
		}

		c.JSON(update.Code, responses.MongoOperationsResultResponse{
			Code:      update.Code,
			Status:    update.Status,
			Database:  update.Database,
			Table:     update.Table,
			Operation: update.Operation,
			Message:   update.Message,
			Query:     rawQuery,
		})
	})

	// Delete record by ID
	mongoApi.DELETE("/db/:db_name/:table_name/delete/:mongo_id", func(c *gin.Context) {
		dbName := c.Param("db_name")
		tableName := c.Param("table_name")
		mongoId := c.Param("mongo_id")

		deleteById, deleteByIdErr := mongoDb.DB(dbName).Table(tableName).DeleteById(mongoId)

		if deleteByIdErr.Error != "" {
			c.JSON(deleteByIdErr.Code, responses.GenericErrorResponse{
				Code:     deleteByIdErr.Code,
				Status:   deleteByIdErr.Status,
				Error:    deleteByIdErr.Error,
				Database: deleteByIdErr.Database,
				Table:    deleteByIdErr.Table,
			})
			return
		}

		c.JSON(deleteById.Code, responses.MongoOperationsResultResponse{
			Code:      deleteById.Code,
			Status:    deleteById.Status,
			Database:  deleteById.Database,
			Table:     deleteById.Table,
			Operation: deleteById.Operation,
			Message:   deleteById.Message,
		})
	})

	// delete by query
	mongoApi.DELETE("/db/:db_name/:table_name/delete-where", func(c *gin.Context) {
		dbName := c.Param("db_name")
		tableName := c.Param("table_name")

		var andQuery [][]interface{}

		if c.Query("query_and") != "" {
			andQuery = helpers.ParseQuery(c.Query("query_and"))
		}

		var orQuery [][]interface{}

		if c.Query("query_or") != "" {
			orQuery = helpers.ParseQuery(c.Query("query_or"))
		}

		mongoDb.ResetQuery()
		delete, deleteErr :=
			mongoDb.
				DB(dbName).
				Table(tableName).
				AndAll(andQuery).
				OrAll(orQuery).
				Delete()

		var rawQuery interface{}
		if deleteErr.Query != "" {
			rawQuery = json.RawMessage(deleteErr.Query)
		}
		if delete.Query != "" {
			rawQuery = json.RawMessage(delete.Query)
		}

		if deleteErr.Error != "" {
			c.JSON(deleteErr.Code, responses.GenericErrorResponse{
				Code:     deleteErr.Code,
				Status:   deleteErr.Status,
				Error:    deleteErr.Error,
				Database: deleteErr.Database,
				Table:    deleteErr.Table,
				Query:    rawQuery,
			})
			return
		}

		c.JSON(delete.Code, responses.MongoOperationsResultResponse{
			Code:      delete.Code,
			Status:    delete.Status,
			Database:  delete.Database,
			Table:     delete.Table,
			Operation: delete.Operation,
			Message:   delete.Message,
			Query:     rawQuery,
		})
	})

	mongoApi.Run(apiHost + ":" + apiPort)
}
