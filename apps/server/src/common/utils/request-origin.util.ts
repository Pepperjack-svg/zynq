import { Request } from 'express';

function getHeaderValue(
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

function getFirstListValue(value: string | undefined): string | undefined {
  return value?.split(',')[0]?.trim();
}

export function getRequestOrigin(req: Request): string | null {
  const forwardedHost = getFirstListValue(
    getHeaderValue(req.headers['x-forwarded-host']),
  );
  const host =
    forwardedHost || getFirstListValue(getHeaderValue(req.headers.host));

  if (!host) {
    return null;
  }

  const forwardedProto = getFirstListValue(
    getHeaderValue(req.headers['x-forwarded-proto']),
  );
  const protocol = forwardedProto || req.protocol || 'http';

  return `${protocol}://${host}`;
}
