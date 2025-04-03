package testcases

var Test_5_parse_responses = []RequestInfo{
	{Url: "http://localhost:8080/1", Method: "GET", Status: "Not Found", StatusCode: 404},
	{Url: "http://localhost:8080/2", Method: "GET"},
	{Url: "http://localhost:8080/3", Method: "GET", Status: "OK", StatusCode: 200},
}
