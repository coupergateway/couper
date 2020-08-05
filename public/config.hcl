server "ferndrang" {
   files {
       document_root = "."
   }

   spa {
       bootstrap_file = "bs.html"
       paths = [
           "/foo/**"
       ]
   }
}