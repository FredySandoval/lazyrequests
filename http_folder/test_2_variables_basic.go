package testcases

var Test_2_variables_basic = []RequestInfo{
	{Url: "http://localhost:8080/version1/1", Method: "GET"},
	{Url: "http://localhost:8080/version1/2", Method: "GET"},
	{Url: "http://localhost:8080/version1/3", Method: "POST"},
	{Url: "http://example.com/version2/4", Method: "GET"},
	{Url: "http://example.com/version2/5", Method: "GET"},
	{Url: "http://example.com/version2/6", Method: "GET"},
	{Url: "http://example.com/version2/7?page=7&pageSize=7", Method: "GET"},
	{Url: "http://example.com/version2?page=8&pageSize=8", Method: "GET"},
	{Url: "http://example.com/version2?page=2&pageSize=9", Method: "GET"},
	{Url: "http://example.com/version1?page=2&pageSize=10", Method: "GET"},
	{Url: "http://example.com/version1?page=2&pageSize=11", Method: "GET"},
	{Url: "http://example.com/version1?page=12&pageSize=12", Method: "POST"},
	{Url: "http://example.com/version1/?page=13&pageSize=13", Method: "GET"},
}
