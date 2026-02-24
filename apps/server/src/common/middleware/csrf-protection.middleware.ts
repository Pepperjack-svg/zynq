import { Injectable, NestMiddleware, ForbiddenException } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { Request, Response, NextFunction } from 'express';

@Injectable()
export class CsrfProtectionMiddleware implements NestMiddleware {
  constructor(private readonly configService: ConfigService) {}

  private getAllowedOrigins(): Set<string> {
    const configured = (
      this.configService.get<string>('CORS_ORIGIN') ||
      this.configService.get<string>('FRONTEND_URL') ||
      ''
    )
      .split(',')
      .map((origin) => origin.trim())
      .filter(Boolean);
    return new Set(configured);
  }

  private getRequestOrigin(req: Request): string | null {
    const origin = req.headers.origin;
    if (typeof origin === 'string' && origin) return origin;

    const referer = req.headers.referer;
    if (typeof referer === 'string' && referer) {
      try {
        return new URL(referer).origin;
      } catch {
        return null;
      }
    }

    return null;
  }

  use(req: Request, _res: Response, next: NextFunction) {
    const method = req.method.toUpperCase();
    const isSafeMethod =
      method === 'GET' || method === 'HEAD' || method === 'OPTIONS';
    if (isSafeMethod) {
      next();
      return;
    }

    // Only enforce for authenticated cookie-based requests.
    if (!req.cookies?.jid) {
      next();
      return;
    }

    const allowedOrigins = this.getAllowedOrigins();
    const requestOrigin = this.getRequestOrigin(req);
    if (!requestOrigin || !allowedOrigins.has(requestOrigin)) {
      throw new ForbiddenException('CSRF validation failed');
    }

    next();
  }
}
