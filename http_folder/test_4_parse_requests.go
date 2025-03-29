package testcases

var Test_4_parse_requests = []RequestInfo{
	{Url: "http://localhost:8080/1", Method: "GET"},
	{Url: "http://localhost:8080/2", Method: "GET"},
	{Url: "http://localhost:8080/3", Method: "POST", Body: "{\"Hello\":\"World\",\"Hello2\":\"World2\"}\n\n"},
}
