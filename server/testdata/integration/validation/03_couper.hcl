server "couper" {
  endpoint "/q" {
	proxy {
      url = "http://localhost:8080/x"
	}
  }

  endpoint "/x" {
	response {
		status = 204
	}
	custom_log_fields = {
		TEST = request.query
	}
  }
}
