server "api" {
  api {
    endpoint "/" {
      proxy {
        backend "anything" {
          remove_request_headers = [ "aeb_del", "CaseIns", request.query.xyz[0] ]
          set_request_headers = {
            aeb_string = "str"
            aeb_multi = ["str1", "str2"]
            aeb_a_and_b = "A&B"
            aeb_noop = request.headers.noop
            aeb_null = null
            aeb_empty = ""
            xxx = ["aaa", "bbb"]
            "${request.query.aeb[0]}" = "aeb"
          }
          add_request_headers = {
            aeb_string = "str"
           aeb_multi = ["str3", "str4"]
            aeb_a_and_b = "A&B"
            aeb_noop = request.headers.noop
            aeb_null = null
            aeb_empty = ""
            "${request.query.aeb[0]}" = "aeb"
          }

          remove_response_headers = [ "Remove-Me-2" ]
          set_response_headers = {
            "Set-Me-2" = "s2"
          }
          add_response_headers = {
            "Add-Me-2" = "a2"
          }
        }
      }
    }
  }
}

definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "anything" {
    origin = env.COUPER_TEST_BACKEND_ADDR

    remove_request_headers = [ "def_del" ]
    set_request_headers = {
      def_string = "str"
      def_multi = ["str1", "str2"]
      def_a_and_b = "A&B"
      def_noop = request.headers.noop
      def_null = null
      def_empty = ""
      xxx = "ddd"
      "${request.query.def[0]}" = "def"
      foo = request.query.foo[0]
    }
    add_request_headers = {
      def_string = "str"
      def_multi = ["str3", "str4"]
      def_a_and_b = "A&B"
      def_noop = request.headers.noop
      def_null = null
      def_empty = ""
      xxx = "eee"
      "${request.query.def[0]}" = "def"
    }

    remove_response_headers = [ "remove-me-1" ]
    set_response_headers = {
      "set-me-1" = "s1"
    }
    add_response_headers = {
      "add-me-1" = "a1"
    }
  }
}
