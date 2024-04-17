#notarize {
#  path = ".macos/couper.dmg"
#  bundle_id = "app.com.xxx.couper"
#  staple = true
#}

notarize {
  path = ".macos/couper.zip"
  bundle_id = "binary.com.xxx.couper"
}

apple_id {
  username = "marcel.ludwig@xxx"
  password = "@env:AC_PASSWORD"
}
