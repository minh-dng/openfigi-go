package openfigi

var httpStatusMap = map[int]string{
	400: "Bad Request. \n" +
		"Invalid Payload.\n" +
		"For enum request, with query key invalid.\n" +
		"For mapping request, with post data not an array.\n" +
		"For searching request, with post data not a plain object.",
	401: "Unauthorized. Invalid API Key",
	404: "Invalid URL",
	405: "Invalid HTTP method",
	406: "Unsupported 'Accept' type",
	413: "Payload too large.\n" +
		"Mapping request with too many items (> 10 for non-apikey, > 100 for apikey)",
	415: "Invalid 'Content-Type' header",
	429: "Rate limit exceeded.\n" +
		"Check headers for details: limit (X-RateLimit-Limit), " +
		"current usage (X-RateLimit-Remaining) and " +
		"reset time (if available) (X-RateLimit-Reset)",
	500: "Internal Server Error",
	503: "Service Unavailable",
}
