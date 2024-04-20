# Ginified MongoDB API Handler (Go)

## Overview

This Go project provides a RESTful API for interacting with a MongoDB server. It utilizes the Gin web framework for building the API endpoints and MongoDB Go driver for database operations.

## Features

- **CRUD Operations**: Perform Create, Read, Update, and Delete operations on MongoDB collections via HTTP requests.
- **JSON Handling**: Send and receive JSON data seamlessly with Gin's built-in JSON support.

## Installation

1. Ensure you have Go installed on your machine. If not, you can download and install it from the [official Go website](https://golang.org/dl/).
2. Clone this repository to your local machine:

    ```bash
    git clone https://github.com/alexanderthegreat96/mongo-db-api-go.git
    ```

3. Install project dependencies:

    ```bash
    go mod tidy
    ```

4. Set up your MongoDB server and ensure it's running.