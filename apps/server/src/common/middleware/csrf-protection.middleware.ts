import {
  Injectable,
  NestMiddleware,
  ForbiddenException,
  Logger,
} from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { Request, Response, NextFunction } from 'express';

@Injectable()
export class CsrfProtectionMiddleware implements NestMiddleware {
  private readonly logger = new Logger(CsrfProtectionMiddleware.name);
  private readonly allowedOrigins: Set<string>;

  constructor(private readonly configService: ConfigService) {
    const configured = (
      this.configService.get<string>('CORS_ORIGIN') ||
      this.configService.get<string>('FRONTEND_URL') ||
      ''
    )
      .split(',')
      .map((origin) => origin.trim())
      .filter(Boolean);
    this.allowedOrigins = new Set(configured);
    if (this.allowedOrigins.size === 0) {
      this.logger.warn(
        'No allowed CSRF origins configured (CORS_ORIGIN/FRONTEND_URL). Authenticated state-changing requests may be rejected.',
      );
    }
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

    const requestOrigin = this.getRequestOrigin(req);
    if (!requestOrigin || !this.allowedOrigins.has(requestOrigin)) {
      this.logger.warn(
        `Blocked CSRF request: ${method} ${req.originalUrl || req.url} origin=${requestOrigin || 'missing'}`,
      );
      throw new ForbiddenException('CSRF validation failed');
    }

    next();
  }
}
