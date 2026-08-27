server "protected" {
  hosts = ["*:8080"]

  api {
    # The documented pattern: an AuthZEN resource search filters an upstream listing down
    # to the permitted subset. Both callouts are independent and run in parallel.
    endpoint "/documents" {
      request "search" {
        url = "http://127.0.0.1:8081/access/v1/search/resource"
        json_body = {
          subject  = { type = "identity", id = "clark.kent" }
          action   = { name = "can_read" }
          resource = { type = "document" }
        }
        expected_status = [200]
      }

      request "list" {
        url = "http://127.0.0.1:8081/documents"
      }

      response {
        json_body = [
          for doc in backend_responses.list.json_body.documents : doc
          if contains([for r in backend_responses.search.json_body.results : r.id], doc.id)
        ]
      }
    }
  }
}

server "services" {
  hosts = ["*:8081"]

  api {
    endpoint "/access/v1/search/resource" {
      response {
        json_body = {
          results = [
            { type = "document", id = "roadmap" },
            { type = "document", id = "budget" },
          ]
        }
      }
    }

    endpoint "/documents" {
      response {
        json_body = {
          documents = [
            { id = "roadmap", title = "Roadmap" },
            { id = "secret", title = "Secret" },
            { id = "budget", title = "Budget" },
          ]
        }
      }
    }
  }
}
