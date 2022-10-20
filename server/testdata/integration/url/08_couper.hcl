server {
  endpoint "/p1/{x}/{y}" {
    response {
      headers = {
        Match = "/p1/{x}/{y}"
      }
    }
  }
  endpoint "/**" {
    response {
      headers = {
        Match = "/**"
      }
    }
  }
  endpoint "/p2/{x}/{y}" {
    response {
      headers = {
        Match = "/p2/{x}/{y}"
      }
    }
  }
  endpoint "/p2/**" {
    response {
      headers = {
        Match = "/p2/**"
      }
    }
  }
  endpoint "/p3/{x}/{y}" {
    response {
      headers = {
        Match = "/p3/{x}/{y}"
      }
    }
  }
  endpoint "/p3/{x}" {
    response {
      headers = {
        Match = "/p3/{x}"
      }
    }
  }
  endpoint "/p3/**" {
    response {
      headers = {
        Match = "/p3/**"
      }
    }
  }
}
