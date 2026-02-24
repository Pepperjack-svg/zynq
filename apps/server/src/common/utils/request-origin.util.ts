import { Request } from 'express';

export function getHeaderValue(
  value: string | string[] | undefined,
): string | undefined {
  if (Array.isArray(value)) {
    return value[0];
  }
  if (typeof value === 'string') {
    return value;
  }
  return undefined;
}

export function getFirstListValue(
  value: string | undefined,
): string | undefined {
  return value?.split(',')[0]?.trim();
}

function normalizeConfiguredOrigin(origin?: string): string | null {
  if (!origin) return null;
  const trimmed = origin.trim().replace(/\/+$/, '');
  if (!trimmed) return null;
  try {
    const url = new URL(trimmed);
    if (url.protocol !== 'http:' && url.protocol !== 'https:') return null;
    return `${url.protocol}//${url.host}`;
  } catch {
    return null;
  }
}

function isAllowedOrigin(origin: string): boolean {
  const raw = process.env.ALLOWED_ORIGINS;
  if (!raw?.trim()) return true;

  const allowedOrigins = raw
    .split(',')
    .map((value) => normalizeConfiguredOrigin(value))
    .filter((value): value is string => Boolean(value));

  if (allowedOrigins.length === 0) return false;
  return allowedOrigins.includes(origin);
}

export function getRequestOrigin(req: Request): string | null {
  const configuredFrontend = normalizeConfiguredOrigin(
    process.env.FRONTEND_URL,
  );
  if (configuredFrontend) {
    return configuredFrontend;
  }

  const trustProxy = process.env.TRUST_PROXY === 'true';

  const forwardedHost = trustProxy
    ? getFirstListValue(getHeaderValue(req.headers['x-forwarded-host']))
    : undefined;
  const host =
    forwardedHost || getFirstListValue(getHeaderValue(req.headers.host));

  if (!host) {
    return null;
  }

  const forwardedProto = getFirstListValue(
    getHeaderValue(req.headers['x-forwarded-proto']),
  )?.toLowerCase();
  const protocol =
    (trustProxy && (forwardedProto === 'http' || forwardedProto === 'https')
      ? forwardedProto
      : undefined) ||
    req.protocol ||
    'http';

  const requestOrigin = `${protocol}://${host}`;
  if (!isAllowedOrigin(requestOrigin)) {
    return null;
  }

  return requestOrigin;
}
