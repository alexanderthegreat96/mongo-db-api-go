## Refactor API
 - handle logic separately
 - as well as the response
 - write tests for the api

## Driver
 - improve query handling
 - add the ability to use ordered hashmaps, even if slow, depending on the user need
 - write tests

## Query Parsing
 - add autoConvertType parameter -> default to false
 - write tests -> test query parsing for all operators
 - update docs
 - test queries extensively
 - ensure that single where's are not placed under $or or $and, unless 2 or more are present
 - check the ai generate improved function as it does not work properly

## Ask AI for more improvements
 - apply these improvements
 - maybe store users in a sqlite DB?
 - add a cli tool for handling the user creation
 - should probably run the API as a daemon as opposed to as standalone
 - implement request logging