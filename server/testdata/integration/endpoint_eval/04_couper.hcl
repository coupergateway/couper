server "api" {
  api {
    endpoint "/" {
      proxy {
        backend "anything" {
          remove_query_params = [ "aeb_del", "CaseIns", req.headers.xyz ]
          set_query_params = {
            aeb_string = "str"
            aeb_multi = ["str1", "str2"]
            aeb_a_and_b = "A&B"
            aeb_noop = req.query.noop
            aeb_null = null
            aeb_empty = ""
            xxx = ["yyy", "xxx"]
            xxx = ["aaa", "bbb"]
            "${req.headers.aeb}" = "aeb"
          }
          add_query_params = {
            aeb_string = "str"
            aeb_multi = ["str3", "str4"]
            aeb_a_and_b = "A&B"
            aeb_noop = req.query.noop
            aeb_null = null
            aeb_empty = ""
            "${req.headers.aeb}" = "aeb"
          }
        }
      }

      remove_query_params = [ "ae_del" ]
      set_query_params = {
        ae_string = "str"
        ae_multi = ["str1", "str2"]
        ae_a_and_b = "A&B"
        ae_noop = req.query.noop
        ae_null = null
        ae_empty = ""
        xxx = "zzz"
        "${req.headers.ae}" = "ae"
      }
      add_query_params = {
        ae_string = "str"
        ae_multi = ["str3", "str4"]
        ae_a_and_b = "A&B"
        ae_noop = req.query.noop
        ae_null = null
        ae_empty = ""
        xxx = "ccc"
        "${req.headers.ae}" = "ae"
      }
    }
  }

# TODO: free-endpoints
#   endpoint "/free/endpoint" {
#     backend {
#       origin = "https://w11w.de"
#       hostname = "w11w.de"

#       remove_query_params = [ "feb_del" ]
#       set_query_params = {
#         feb_string = "str"
#         feb_multi = ["str1", "str2"]
#         feb_a_and_b = "A&B"
#         feb_noop = req.query.noop
#         feb_null = null
#         feb_empty = ""
#       }
#       add_query_params = {
#         feb_string = "str"
#         feb_multi = ["str3", "str4"]
#         feb_a_and_b = "A&B"
#         feb_noop = req.query.noop
#         feb_null = null
#         feb_empty = ""
#       }
#     }

#     remove_query_params = [ "fe_del" ]
#     set_query_params = {
#       fe_String = "str"
#       fe_multi = ["str1", "str2"]
#       fe_a_and_b = "A&B"
#       fe_noop = req.query.noop
#       fe_null = null
#       fe_empty = ""
#     }
#     add_query_params = {
#       fe_String = "str"
#       fe_multi = ["str3", "str4"]
#       fe_a_and_b = "A&B"
#       fe_noop = req.query.noop
#       fe_null = null
#       fe_empty = ""
#     }
#   }
}

definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "anything" {
    origin = env.COUPER_TEST_BACKEND_ADDR

    remove_query_params = [ "def_del" ]
    set_query_params = {
      def_string = "str"
      def_multi = ["str1", "str2"]
      def_a_and_b = "A&B"
      def_noop = req.query.noop
      def_null = null
      def_empty = ""
      xxx = "ddd"
      "${req.headers.def}" = "def"
      foo = req.headers.foo
    }
    add_query_params = {
      def_string = "str"
      def_multi = ["str3", "str4"]
      def_a_and_b = "A&B"
      def_noop = req.query.noop
      def_null = null
      def_empty = ""
      xxx = "eee"
      "${req.headers.def}" = "def"
    }
  }
}
