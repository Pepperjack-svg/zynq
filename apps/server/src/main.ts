import { NestFactory } from '@nestjs/core';
import { ValidationPipe, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import * as cookieParser from 'cookie-parser';
import { AppModule } from './app.module';
import { GlobalExceptionFilter } from './common/filters/http-exception.filter';

async function bootstrap() {
  const logger = new Logger('Bootstrap');
  const app = await NestFactory.create(AppModule);
  const configService = app.get(ConfigService);
  const trustProxy = configService.get('TRUST_PROXY') === 'true';

  if (trustProxy) {
    app.getHttpAdapter().getInstance().set('trust proxy', 1);
  }

  // Global prefix
  app.setGlobalPrefix('api/v1');

  // Cookie parser
  app.use(cookieParser());

  const corsOriginsFromEnv = (
    configService.get<string>('CORS_ORIGIN') ||
    configService.get<string>('FRONTEND_URL') ||
    ''
  )
    .split(',')
    .map((origin) => origin.trim())
    .filter(Boolean);

  const allowedOrigins = new Set(corsOriginsFromEnv);

  // CORS — only allow configured origins when credentials are enabled
  app.enableCors({
    origin: (origin, callback) => {
      // Non-browser requests (curl, health checks)
      if (!origin) {
        callback(null, true);
        return;
      }

      if (allowedOrigins.has(origin)) {
        callback(null, true);
        return;
      }

      callback(null, false);
    },
    credentials: true,
  });

  // Global exception filter
  app.useGlobalFilters(new GlobalExceptionFilter());

  // Global validation pipe
  app.useGlobalPipes(
    new ValidationPipe({
      whitelist: true,
      transform: true,
      forbidNonWhitelisted: true,
    }),
  );

  const port = configService.get('PORT') || 4000;
  await app.listen(port);

  logger.log(`zynqCloud backend running on http://localhost:${port}`);
  logger.log(`API available at http://localhost:${port}/api/v1`);
  logger.log(`Health check at http://localhost:${port}/api/v1/health`);
}

bootstrap();
