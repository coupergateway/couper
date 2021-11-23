#notarize {
#  path = ".macos/couper.dmg"
#  bundle_id = "com.avenga.couper"
#  staple = true
#}

notarize {
  path = ".macos/couper.zip"
  bundle_id = "com.avenga.couper"
}

apple_id {
  username = "marcel.ludwig@avenga.com"
  password = "@env:AC_PASSWORD"
}
