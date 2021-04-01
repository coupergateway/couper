server "api-ac-1" {
  hosts = ["example.com:8080"]

  endpoint "/v1" {
    response {
      body = "s1"
    }
  }
}

server "api-ac-2" {
  hosts = ["v1.example.com:8080"]

  endpoint "/" {
    response {
      body = "s2"
    }
  }
}
