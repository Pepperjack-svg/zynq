import { Request } from 'express';
import {
  getHeaderValue,
  getFirstListValue,
  getRequestOrigin,
} from './request-origin.util';

function makeRequest(
  headers: Record<string, string | string[] | undefined>,
  protocol?: string,
): Request {
  return {
    headers,
    protocol,
  } as Request;
}

describe('request-origin util', () => {
  const originalFrontendUrl = process.env.FRONTEND_URL;
  const originalAllowedOrigins = process.env.ALLOWED_ORIGINS;
  const originalTrustProxy = process.env.TRUST_PROXY;

  beforeEach(() => {
    delete process.env.FRONTEND_URL;
    delete process.env.ALLOWED_ORIGINS;
    delete process.env.TRUST_PROXY;
  });

  afterAll(() => {
    process.env.FRONTEND_URL = originalFrontendUrl;
    process.env.ALLOWED_ORIGINS = originalAllowedOrigins;
    process.env.TRUST_PROXY = originalTrustProxy;
  });

  describe('getHeaderValue', () => {
    it('returns first element when header value is an array', () => {
      expect(getHeaderValue(['a', 'b'])).toBe('a');
    });

    it('returns header value when it is a string', () => {
      expect(getHeaderValue('value')).toBe('value');
    });

    it('returns undefined for missing value', () => {
      expect(getHeaderValue(undefined)).toBeUndefined();
    });
  });

  describe('getFirstListValue', () => {
    it('returns first comma separated value', () => {
      expect(getFirstListValue('a, b, c')).toBe('a');
    });

    it('returns undefined when input is undefined', () => {
      expect(getFirstListValue(undefined)).toBeUndefined();
    });
  });

  describe('getRequestOrigin', () => {
    it('prefers configured FRONTEND_URL over request headers', () => {
      process.env.FRONTEND_URL = 'https://configured.example.com/';
      const req = makeRequest({ host: 'malicious.example.com' }, 'http');

      expect(getRequestOrigin(req)).toBe('https://configured.example.com');
    });

    it('uses x-forwarded-host over host when TRUST_PROXY=true', () => {
      process.env.TRUST_PROXY = 'true';
      const req = makeRequest(
        {
          'x-forwarded-host': 'proxy.example:80',
          host: 'ignored.example:3000',
        },
        'https',
      );

      expect(getRequestOrigin(req)).toBe('https://proxy.example:80');
    });

    it('returns null when neither host nor forwarded host exists', () => {
      const req = makeRequest({}, 'https');
      expect(getRequestOrigin(req)).toBeNull();
    });

    it('uses x-forwarded-proto when TRUST_PROXY=true and value is valid', () => {
      process.env.TRUST_PROXY = 'true';
      const req = makeRequest(
        {
          host: 'example.com',
          'x-forwarded-proto': 'HTTPS',
        },
        'http',
      );

      expect(getRequestOrigin(req)).toBe('https://example.com');
    });

    it('falls back to req.protocol when forwarded proto is invalid', () => {
      process.env.TRUST_PROXY = 'true';
      const req = makeRequest(
        {
          host: 'example.com',
          'x-forwarded-proto': 'javascript',
        },
        'https',
      );

      expect(getRequestOrigin(req)).toBe('https://example.com');
    });

    it('defaults to http when protocol is missing', () => {
      const req = makeRequest({ host: 'example.com' });
      expect(getRequestOrigin(req)).toBe('http://example.com');
    });

    it('rejects request-derived origin when not in ALLOWED_ORIGINS', () => {
      process.env.ALLOWED_ORIGINS = 'https://allowed.example.com';
      const req = makeRequest({ host: 'denied.example.com' }, 'https');

      expect(getRequestOrigin(req)).toBeNull();
    });
  });
});
