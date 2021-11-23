#notarize {
#  path = ".macos/couper.dmg"
#  bundle_id = "app.com.avenga.couper"
#  staple = true
#}

notarize {
  path = ".macos/couper.zip"
  bundle_id = "binary.com.avenga.couper"
}

apple_id {
  username = "marcel.ludwig@avenga.com"
  password = "@env:AC_PASSWORD"
}
