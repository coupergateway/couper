server "api" {
  api {
    endpoint "/" {
      backend "anything" {
        remove_request_headers = [ "aeb_del", "CaseIns", req.query.xyz[0] ]
        set_request_headers = {
          aeb_string = "str"
          aeb_multi = ["str1", "str2"]
          aeb_a_and_b = "A&B"
          aeb_noop = req.headers.noop
          aeb_null = null
          aeb_empty = ""
          xxx = ["yyy", "xxx"]
          xxx = ["aaa", "bbb"]
          "${req.query.aeb[0]}" = "aeb"
        }
        add_request_headers = {
          aeb_string = "str"
          aeb_multi = ["str3", "str4"]
          aeb_a_and_b = "A&B"
          aeb_noop = req.headers.noop
          aeb_null = null
          aeb_empty = ""
          "${req.query.aeb[0]}" = "aeb"
        }
      }

      remove_request_headers = [ "ae_del" ]
      set_request_headers = {
        ae_string = "str"
        ae_multi = ["str1", "str2"]
        ae_a_and_b = "A&B"
        ae_noop = req.headers.noop
        ae_null = null
        ae_empty = ""
        xxx = "zzz"
        "${req.query.ae[0]}" = "ae"
      }
      add_request_headers = {
        ae_string = "str"
        ae_multi = ["str3", "str4"]
        ae_a_and_b = "A&B"
        ae_noop = req.headers.noop
        ae_null = null
        ae_empty = ""
        xxx = "ccc"
        "${req.query.ae[0]}" = "ae"
      }
    }
  }

# TODO: free-endpoints
#   endpoint "/free/endpoint" {
#     backend {
#       origin = "https://w11w.de"
#       hostname = "w11w.de"

#       remove_request_headers = [ "feb_del" ]
#       set_request_headers = {
#         feb_string = "str"
#         feb_multi = ["str1", "str2"]
#         feb_a_and_b = "A&B"
#         feb_noop = req.headers.noop
#         feb_null = null
#         feb_empty = ""
#       }
#       add_request_headers = {
#         feb_string = "str"
#         feb_multi = ["str3", "str4"]
#         feb_a_and_b = "A&B"
#         feb_noop = req.headers.noop
#         feb_null = null
#         feb_empty = ""
#       }
#     }

#     remove_request_headers = [ "fe_del" ]
#     set_request_headers = {
#       fe_String = "str"
#       fe_multi = ["str1", "str2"]
#       fe_a_and_b = "A&B"
#       fe_noop = req.headers.noop
#       fe_null = null
#       fe_empty = ""
#     }
#     add_request_headers = {
#       fe_String = "str"
#       fe_multi = ["str3", "str4"]
#       fe_a_and_b = "A&B"
#       fe_noop = req.headers.noop
#       fe_null = null
#       fe_empty = ""
#     }
#   }
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
      def_noop = req.headers.noop
      def_null = null
      def_empty = ""
      xxx = "ddd"
      "${req.query.def[0]}" = "def"
      foo = req.query.foo[0]
    }
    add_request_headers = {
      def_string = "str"
      def_multi = ["str3", "str4"]
      def_a_and_b = "A&B"
      def_noop = req.headers.noop
      def_null = null
      def_empty = ""
      xxx = "eee"
      "${req.query.def[0]}" = "def"
    }
  }
}
