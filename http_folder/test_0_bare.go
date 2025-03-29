package testcases

type RequestInfo struct {
	Url               string
	Method            string
	CommentIdentifier string
	Body              string
	Status            string
	StatusCode        int
}

var Test_0_bare = []RequestInfo{
	{Url: "http://localhost:8080/v1/comments/1", Method: "GET"},
}
