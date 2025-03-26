// Code generated by smithy-go-codegen DO NOT EDIT.

package ssooidc

import (
	"context"
	"fmt"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Creates and returns access and refresh tokens for clients that are
// authenticated using client secrets. The access token can be used to fetch
// short-lived credentials for the assigned AWS accounts or to access application
// APIs using bearer authentication.
func (c *Client) CreateToken(ctx context.Context, params *CreateTokenInput, optFns ...func(*Options)) (*CreateTokenOutput, error) {
	if params == nil {
		params = &CreateTokenInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "CreateToken", params, optFns, c.addOperationCreateTokenMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*CreateTokenOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type CreateTokenInput struct {

	// The unique identifier string for the client or application. This value comes
	// from the result of the RegisterClientAPI.
	//
	// This member is required.
	ClientId *string

	// A secret string generated for the client. This value should come from the
	// persisted result of the RegisterClientAPI.
	//
	// This member is required.
	ClientSecret *string

	// Supports the following OAuth grant types: Authorization Code, Device Code, and
	// Refresh Token. Specify one of the following values, depending on the grant type
	// that you want:
	//
	// * Authorization Code - authorization_code
	//
	// * Device Code - urn:ietf:params:oauth:grant-type:device_code
	//
	// * Refresh Token - refresh_token
	//
	// This member is required.
	GrantType *string

	// Used only when calling this API for the Authorization Code grant type. The
	// short-lived code is used to identify this authorization request.
	Code *string

	// Used only when calling this API for the Authorization Code grant type. This
	// value is generated by the client and presented to validate the original code
	// challenge value the client passed at authorization time.
	CodeVerifier *string

	// Used only when calling this API for the Device Code grant type. This
	// short-lived code is used to identify this authorization request. This comes from
	// the result of the StartDeviceAuthorizationAPI.
	DeviceCode *string

	// Used only when calling this API for the Authorization Code grant type. This
	// value specifies the location of the client or application that has registered to
	// receive the authorization code.
	RedirectUri *string

	// Used only when calling this API for the Refresh Token grant type. This token is
	// used to refresh short-lived tokens, such as the access token, that might expire.
	//
	// For more information about the features and limitations of the current IAM
	// Identity Center OIDC implementation, see Considerations for Using this Guide in
	// the [IAM Identity Center OIDC API Reference].
	//
	// [IAM Identity Center OIDC API Reference]: https://docs.aws.amazon.com/singlesignon/latest/OIDCAPIReference/Welcome.html
	RefreshToken *string

	// The list of scopes for which authorization is requested. The access token that
	// is issued is limited to the scopes that are granted. If this value is not
	// specified, IAM Identity Center authorizes all scopes that are configured for the
	// client during the call to RegisterClient.
	Scope []string

	noSmithyDocumentSerde
}

type CreateTokenOutput struct {

	// A bearer token to access Amazon Web Services accounts and applications assigned
	// to a user.
	AccessToken *string

	// Indicates the time in seconds when an access token will expire.
	ExpiresIn int32

	// The idToken is not implemented or supported. For more information about the
	// features and limitations of the current IAM Identity Center OIDC implementation,
	// see Considerations for Using this Guide in the [IAM Identity Center OIDC API Reference].
	//
	// A JSON Web Token (JWT) that identifies who is associated with the issued access
	// token.
	//
	// [IAM Identity Center OIDC API Reference]: https://docs.aws.amazon.com/singlesignon/latest/OIDCAPIReference/Welcome.html
	IdToken *string

	// A token that, if present, can be used to refresh a previously issued access
	// token that might have expired.
	//
	// For more information about the features and limitations of the current IAM
	// Identity Center OIDC implementation, see Considerations for Using this Guide in
	// the [IAM Identity Center OIDC API Reference].
	//
	// [IAM Identity Center OIDC API Reference]: https://docs.aws.amazon.com/singlesignon/latest/OIDCAPIReference/Welcome.html
	RefreshToken *string

	// Used to notify the client that the returned token is an access token. The
	// supported token type is Bearer .
	TokenType *string

	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationCreateTokenMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsRestjson1_serializeOpCreateToken{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsRestjson1_deserializeOpCreateToken{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "CreateToken"); err != nil {
		return fmt.Errorf("add protocol finalizers: %v", err)
	}

	if err = addlegacyEndpointContextSetter(stack, options); err != nil {
		return err
	}
	if err = addSetLoggerMiddleware(stack, options); err != nil {
		return err
	}
	if err = addClientRequestID(stack); err != nil {
		return err
	}
	if err = addComputeContentLength(stack); err != nil {
		return err
	}
	if err = addResolveEndpointMiddleware(stack, options); err != nil {
		return err
	}
	if err = addRetry(stack, options); err != nil {
		return err
	}
	if err = addRawResponseToMetadata(stack); err != nil {
		return err
	}
	if err = addRecordResponseTiming(stack); err != nil {
		return err
	}
	if err = addSpanRetryLoop(stack, options); err != nil {
		return err
	}
	if err = addClientUserAgent(stack, options); err != nil {
		return err
	}
	if err = smithyhttp.AddErrorCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = addSetLegacyContextSigningOptionsMiddleware(stack); err != nil {
		return err
	}
	if err = addTimeOffsetBuild(stack, c); err != nil {
		return err
	}
	if err = addUserAgentRetryMode(stack, options); err != nil {
		return err
	}
	if err = addOpCreateTokenValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opCreateToken(options.Region), middleware.Before); err != nil {
		return err
	}
	if err = addRecursionDetection(stack); err != nil {
		return err
	}
	if err = addRequestIDRetrieverMiddleware(stack); err != nil {
		return err
	}
	if err = addResponseErrorMiddleware(stack); err != nil {
		return err
	}
	if err = addRequestResponseLogging(stack, options); err != nil {
		return err
	}
	if err = addDisableHTTPSMiddleware(stack, options); err != nil {
		return err
	}
	if err = addSpanInitializeStart(stack); err != nil {
		return err
	}
	if err = addSpanInitializeEnd(stack); err != nil {
		return err
	}
	if err = addSpanBuildRequestStart(stack); err != nil {
		return err
	}
	if err = addSpanBuildRequestEnd(stack); err != nil {
		return err
	}
	return nil
}

func newServiceMetadataMiddleware_opCreateToken(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "CreateToken",
	}
}
