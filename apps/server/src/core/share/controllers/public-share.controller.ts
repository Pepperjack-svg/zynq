import {
  Controller,
  ForbiddenException,
  Get,
  Headers,
  HttpException,
  HttpStatus,
  Param,
  Req,
  Res,
} from '@nestjs/common';
import { Throttle } from '@nestjs/throttler';
import { Request, Response } from 'express';
import { FileService } from '../../file/file.service';

@Controller('public')
export class PublicShareController {
  private static readonly PASSWORD_WINDOW_MS = 60_000;
  private static readonly PASSWORD_WINDOW_MAX_ATTEMPTS = 10;

  private failedPasswordAttempts = new Map<
    string,
    { attempts: number; blockedUntil?: number }
  >();
  private passwordAttemptWindows = new Map<
    string,
    { startedAt: number; attempts: number }
  >();

  constructor(private fileService: FileService) {}

  @Get('share/:token')
  @Throttle({ default: { limit: 30, ttl: 60000 } })
  async getPublicShare(
    @Param('token') token: string,
    @Req() req: Request,
    @Headers('x-share-password') password?: string,
  ) {
    const key = `${req.ip}:${token}`;
    const current = this.failedPasswordAttempts.get(key);
    const now = Date.now();

    if (password) {
      const windowState = this.passwordAttemptWindows.get(key);
      if (
        !windowState ||
        now - windowState.startedAt > PublicShareController.PASSWORD_WINDOW_MS
      ) {
        this.passwordAttemptWindows.set(key, { startedAt: now, attempts: 1 });
      } else {
        const attempts = windowState.attempts + 1;
        if (attempts > PublicShareController.PASSWORD_WINDOW_MAX_ATTEMPTS) {
          throw new HttpException(
            'Too many password attempts. Retry in 60s',
            HttpStatus.TOO_MANY_REQUESTS,
          );
        }
        this.passwordAttemptWindows.set(key, {
          startedAt: windowState.startedAt,
          attempts,
        });
      }
    } else {
      this.passwordAttemptWindows.delete(key);
    }

    if (password && current?.blockedUntil && current.blockedUntil > now) {
      throw new HttpException(
        'Too many failed password attempts',
        HttpStatus.TOO_MANY_REQUESTS,
      );
    }

    try {
      const result = await this.fileService.getPublicShare(token, password);
      this.failedPasswordAttempts.delete(key);
      if (password) {
        this.passwordAttemptWindows.delete(key);
      }
      return result;
    } catch (error) {
      if (password && error instanceof ForbiddenException) {
        const attempts = (current?.attempts ?? 0) + 1;
        const backoffSeconds = Math.min(300, 2 ** attempts);
        const blockedUntil = now + backoffSeconds * 1000;
        this.failedPasswordAttempts.set(key, { attempts, blockedUntil });
        if (attempts >= 3) {
          throw new HttpException(
            `Too many failed password attempts. Retry in ${backoffSeconds}s`,
            HttpStatus.TOO_MANY_REQUESTS,
          );
        }
      }
      throw error;
    }
  }

  @Get('share/:token/download')
  @Throttle({ default: { limit: 30, ttl: 60000 } })
  async downloadPublicFile(
    @Param('token') token: string,
    @Res() res: Response,
    @Headers('x-share-password') password?: string,
  ) {
    const { data, file } = await this.fileService.downloadPublicFile(
      token,
      password,
    );

    res.set({
      'Content-Type': file.mime_type || 'application/octet-stream',
      'Content-Disposition': `attachment; filename="${encodeURIComponent(file.name)}"; filename*=UTF-8''${encodeURIComponent(file.name)}`,
      'Access-Control-Expose-Headers': 'Content-Disposition',
      'Content-Length': data.length,
    });

    res.send(data);
  }
}
