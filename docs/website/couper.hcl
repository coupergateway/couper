server {
	files {
		document_root = ".output/public"
	}

	files "prod" {
		base_path = "/couper-docs"
		document_root = ".output/public"
	}
}

