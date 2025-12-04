package aegis

type SchemaConfig struct {
	// A function to return a map of global fields to be added to all schemas when creating a record. e.g:
	//  func(ctx context.Context) map[string]any {
	// 		return map[string]any{
	//  		"uuid": uuid.New().String(),
	//  		"created_at": time.Now(),
	//  		"updated_at": time.Now(),
	// 		 }
	//	 }
	// this function will be called during the creation of any schema record.
	// You can also set fields on supported schemas itself.
	AdditionalFields AdditionalFieldsFunc
	// User schema configuration
	User UserSchema
	// Verification schema configuration
	Verification VerificationSchema
	// Session schema configuration
	Session SessionSchema
	// Rate limit schema configuration
	RateLimit RateLimitSchema
}
