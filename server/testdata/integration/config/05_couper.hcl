server "jwt-source" {}
definitions {
  jwt "invalid-source" {
    header = "foo"
    cookie = "bar"
    signature_algorithm = "HS256"
    key = "y0urS3cretT08eU5edF0rC0uPerInThe3xamp1e"
  }
}
