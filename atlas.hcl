data "ent" "schema" {
  path = "./ent/schema"
}

env "local" {
  src = data.ent.schema.url
  dev = "docker://postgres/16/dev?search_path=public"
  migration {
    dir    = "file://migrations?format=golang-migrate"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
