server "api" {
  api {
    endpoint "/" {
      proxy {
        backend "anything" {
          remove_query_params = [ "aeb_del", "CaseIns", request.headers.xyz ]
          set_query_params = {
            aeb_string = "str"
            aeb_multi = ["str1", "str2"]
            aeb_a_and_b = "A&B"
            aeb_noop = request.query.noop
            aeb_null = null
            aeb_empty = ""
            xxx = ["aaa", "bbb"]
            "${request.headers.aeb}" = "aeb"
          }
          add_query_params = {
            aeb_string = "str"
            aeb_multi = ["str3", "str4"]
            aeb_a_and_b = "A&B"
            aeb_noop = request.query.noop
            aeb_null = null
            aeb_empty = ""
            "${request.headers.aeb}" = "aeb"
          }
        }
      }

      remove_query_params = [ "ae_del" ]
      set_query_params = {
        ae_string = "str"
        ae_multi = ["str1", "str2"]
        ae_a_and_b = "A&B"
        ae_noop = request.query.noop
        ae_null = null
        ae_empty = ""
        xxx = "zzz"
        "${request.headers.ae}" = "ae"
      }
      add_query_params = {
        ae_string = "str"
        ae_multi = ["str3", "str4"]
        ae_a_and_b = "A&B"
        ae_noop = request.query.noop
        ae_null = null
        ae_empty = ""
        xxx = "ccc"
        "${request.headers.ae}" = "ae"
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
#         feb_noop = request.query.noop
#         feb_null = null
#         feb_empty = ""
#       }
#       add_query_params = {
#         feb_string = "str"
#         feb_multi = ["str3", "str4"]
#         feb_a_and_b = "A&B"
#         feb_noop = request.query.noop
#         feb_null = null
#         feb_empty = ""
#       }
#     }

#     remove_query_params = [ "fe_del" ]
#     set_query_params = {
#       fe_String = "str"
#       fe_multi = ["str1", "str2"]
#       fe_a_and_b = "A&B"
#       fe_noop = request.query.noop
#       fe_null = null
#       fe_empty = ""
#     }
#     add_query_params = {
#       fe_String = "str"
#       fe_multi = ["str3", "str4"]
#       fe_a_and_b = "A&B"
#       fe_noop = request.query.noop
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
      def_noop = request.query.noop
      def_null = null
      def_empty = ""
      xxx = "ddd"
      "${request.headers.def}" = "def"
      foo = request.headers.foo
    }
    add_query_params = {
      def_string = "str"
      def_multi = ["str3", "str4"]
      def_a_and_b = "A&B"
      def_noop = request.query.noop
      def_null = null
      def_empty = ""
      xxx = "eee"
      "${request.headers.def}" = "def"
    }
  }
}
