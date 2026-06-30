'use strict';
const { SecretsManagerClient, GetSecretValueCommand } = require('@aws-sdk/client-secrets-manager');
const { createHmac } = require('crypto');

const smClient = new SecretsManagerClient({});

// Cache secret value between warm invocations (5-minute TTL)
let _cachedSecret = null;
let _cacheExpiry  = 0;

async function getJwtSecret() {
  const now = Date.now();
  if (_cachedSecret && now < _cacheExpiry) return _cachedSecret;
  const resp = await smClient.send(
    new GetSecretValueCommand({ SecretId: process.env.JWT_SECRET_ARN })
  );
  _cachedSecret = resp.SecretString;
  _cacheExpiry  = now + 5 * 60 * 1000;
  return _cachedSecret;
}

exports.handler = async (event) => {
  const token = (event.authorizationToken || '').replace(/^Bearer\s+/i, '');
  if (!token) return generatePolicy('anonymous', 'Deny', event.methodArn);

  try {
    const secret = await getJwtSecret();
    const [headerB64, payloadB64, signatureB64] = token.split('.');
    if (!headerB64 || !payloadB64 || !signatureB64)
      return generatePolicy('anonymous', 'Deny', event.methodArn);

    const signingInput  = `${headerB64}.${payloadB64}`;
    const expectedSig   = createHmac('sha256', secret).update(signingInput).digest('base64url');

    if (expectedSig !== signatureB64)
      return generatePolicy('anonymous', 'Deny', event.methodArn);

    const payload = JSON.parse(Buffer.from(payloadB64, 'base64url').toString());
    const now     = Math.floor(Date.now() / 1000);
    if (payload.exp && payload.exp < now)
      return generatePolicy(payload.sub || 'anonymous', 'Deny', event.methodArn);

    const policy = generatePolicy(payload.sub || 'user', 'Allow', event.methodArn);
    policy.context = {
      userId:   payload.sub   || '',
      tenantId: payload.tid   || '',
      email:    payload.email || '',
      roles:    Array.isArray(payload.roles) ? payload.roles.join(',') : '',
    };
    return policy;

  } catch (err) {
    console.error('Authorizer error:', err.message);
    return generatePolicy('anonymous', 'Deny', event.methodArn);
  }
};

function generatePolicy(principalId, effect, resource) {
  return {
    principalId,
    policyDocument: {
      Version: '2012-10-17',
      Statement: [{ Action: 'execute-api:Invoke', Effect: effect, Resource: resource }],
    },
  };
}
