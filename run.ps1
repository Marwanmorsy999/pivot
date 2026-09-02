$env:CGO_ENABLED = "1"
$cmd = "go"; $a = @("run", "./cmd/pivot") + $args; & $cmd @a
