server {
  endpoint "/" {
    request "aaa" {
      url = "http://localhost:8083/aaa"
      form_body = {
        a = backend_responses.a.status
      }
    }
    request "aa" {
      url = "http://localhost:8083/aa"
      form_body = {
        aaa = backend_responses.aaa.status
      }
    }
    request "a" {
      url = "http://localhost:8083/a"
      form_body = {
        aa = backend_responses.aa.status
      }
    }
    response {
      json_body = {
        a = backend_responses.a.status
      }
    }
  }
}
