source = ["./.macos/Couper.app/Contents/MacOS/couper"]
bundle_id = "com.avenga.couper"

apple_id {
  username = "marcel.ludwig@avenga.com"
  password = "@env:AC_PASSWORD"
}

sign {
  application_identity = "4b8fa10ccb8f16f9f464385768d82645831f4644"
  entitlements_file = "./.macos/entitlements.plist"
}
