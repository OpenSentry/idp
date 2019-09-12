Dear {{ .Email }}

You are invited to join {{ .IdentityProvider }} with the following permissions:

{{ if .Scopes }}
  {{ range $key, $scope := .Scopes }}
    {{ $scope.Title }}
    {{ $scope.Description }}
    {{ $scope.Name }}

  {{ end }}
{{end}}

I recommend you to follow:

{{ if .Follows }}
  {{ range $key, $follow := .Follows }}
    {{ $follow.Name }}
    {{ $follow.Introduction }}
    Read more: {{ $follow.PublicProfileUrl }}

  {{ end }}
{{ end }}

To accept or decline the invitation please visit this link: {{ .InvitationUrl }}

Kind Regards,
{{ .OnBehalfOf }}
