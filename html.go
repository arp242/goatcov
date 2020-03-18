package main

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title></title>
	{{/*
	<link rel="icon" type="image/png" href="{% base64 ./favicon.png %}">
	*/}}
	<style>
		table   { border-collapse: collapse; margin-bottom: 1em; }
		caption { white-space: nowrap; font-weight: bold; }
		td      { padding: .2em; border: 1px solid #666; }
	</style>
</head>

<body>
	<p>Total: {{.Overview.Coverage | float}}%</p>

	{{range $f := .Overview.Files}}
		<table>
			<caption>{{$f.Name}} <span>{{$f.Coverage | float}}%</span><caption>
			{{range $fn := $f.Funcs}}
				<tr>
					<td>{{$fn.Name}}</td>
					<td>{{$fn.Coverage | float}}%</td>
				</tr>
			{{- end}}
		</table>
	{{end}}
</body>
</html>
`
