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
  private failedPasswordAttempts = new Map<
    string,
    { attempts: number; blockedUntil?: number }
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

    if (password && current?.blockedUntil && current.blockedUntil > now) {
      throw new HttpException(
        'Too many failed password attempts',
        HttpStatus.TOO_MANY_REQUESTS,
      );
    }

    try {
      const result = await this.fileService.getPublicShare(token, password);
      this.failedPasswordAttempts.delete(key);
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
