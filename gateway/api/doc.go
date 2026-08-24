package api

//	@title			Radio API Gateway
//	@version		1.0
//	@description	Public API gateway. Authentication is cookie-based (HttpOnly) and also
//	@description	accepts a Bearer token. The gateway validates tokens via the user-service
//	@description	and injects a trusted X-Owner-ID header before proxying to the content-service.
//	@BasePath		/
//	@schemes		http
//	@securityDefinitions.apikey	bearerAuth
//	@in				header
//	@name			Authorization
