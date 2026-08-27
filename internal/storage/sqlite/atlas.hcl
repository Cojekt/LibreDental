env "main" {
  src = "file://schema"
  dev = "sqlite://file?mode=memory&_fk=1"
  migration {
    dir    = "file://migrations"
    format = goose
  }
}

env "audit" {
  src = "file://audit_schema"
  dev = "sqlite://file?mode=memory&_fk=1"
  migration {
    dir    = "file://audit_migrations"
    format = goose
  }
}
