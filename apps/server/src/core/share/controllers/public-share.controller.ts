import { Controller, Get, Headers, Param, Res } from '@nestjs/common';
import { Throttle } from '@nestjs/throttler';
import { Response } from 'express';
import { FileService } from '../../file/file.service';

@Controller('public')
export class PublicShareController {
  constructor(private fileService: FileService) {}

  @Get('share/:token')
  @Throttle({ default: { limit: 60, ttl: 60000 } })
  async getPublicShare(
    @Param('token') token: string,
    @Headers('x-share-password') password?: string,
  ) {
    return this.fileService.getPublicShare(token, password);
  }

  @Get('share/:token/download')
  @Throttle({ default: { limit: 30, ttl: 60000 } })
  async downloadPublicFile(
    @Param('token') token: string,
    @Headers('x-share-password') password: string | undefined,
    @Res() res: Response,
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
