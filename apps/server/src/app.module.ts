import { Module, NestModule, MiddlewareConsumer } from '@nestjs/common';
import { ConfigModule, ConfigService } from '@nestjs/config';
import { TypeOrmModule } from '@nestjs/typeorm';
import { ThrottlerModule, ThrottlerGuard } from '@nestjs/throttler';
import { APP_GUARD } from '@nestjs/core';
import { CoreModule } from './core/core.module';
import { EmailModule } from './integrations/email/email.module';
import { HealthModule } from './health/health.module';
import { LoggingMiddleware } from './common/middleware/logging.middleware';

function validateEnv(config: Record<string, unknown>) {
  const requiredKeys = [
    'DATABASE_HOST',
    'DATABASE_PORT',
    'DATABASE_USER',
    'DATABASE_PASSWORD',
    'DATABASE_NAME',
    'JWT_SECRET',
    'FILE_ENCRYPTION_MASTER_KEY',
  ];

  for (const key of requiredKeys) {
    const value = String(config[key] ?? '').trim();
    if (!value) {
      throw new Error(`Missing required environment variable: ${key}`);
    }
  }

  const databasePort = Number(config.DATABASE_PORT);
  if (!Number.isInteger(databasePort) || databasePort <= 0) {
    throw new Error('DATABASE_PORT must be a positive integer');
  }

  const rateLimitTtl = Number(config.RATE_LIMIT_TTL ?? 60000);
  const rateLimitMax = Number(config.RATE_LIMIT_MAX ?? 100);
  if (!Number.isInteger(rateLimitTtl) || rateLimitTtl <= 0) {
    throw new Error('RATE_LIMIT_TTL must be a positive integer');
  }
  if (!Number.isInteger(rateLimitMax) || rateLimitMax <= 0) {
    throw new Error('RATE_LIMIT_MAX must be a positive integer');
  }

  const jwtSecret = String(config.JWT_SECRET);
  if (jwtSecret.length < 32) {
    throw new Error('JWT_SECRET must be at least 32 characters');
  }

  const keyRaw = String(config.FILE_ENCRYPTION_MASTER_KEY);
  const key = Buffer.from(keyRaw, 'base64');
  if (key.length !== 32) {
    throw new Error(
      'FILE_ENCRYPTION_MASTER_KEY must be base64 encoded 32-byte key',
    );
  }

  return {
    ...config,
    DATABASE_PORT: databasePort,
    RATE_LIMIT_TTL: rateLimitTtl,
    RATE_LIMIT_MAX: rateLimitMax,
  };
}

@Module({
  imports: [
    ConfigModule.forRoot({
      isGlobal: true,
      envFilePath: '.env',
      validate: validateEnv,
    }),
    TypeOrmModule.forRootAsync({
      imports: [ConfigModule],
      useFactory: (configService: ConfigService) => ({
        type: 'postgres',
        host: configService.get('DATABASE_HOST'),
        port: configService.get('DATABASE_PORT'),
        username: configService.get('DATABASE_USER'),
        password: configService.get('DATABASE_PASSWORD'),
        database: configService.get('DATABASE_NAME'),
        entities: [__dirname + '/**/*.entity{.ts,.js}'],
        synchronize: configService.get('NODE_ENV') === 'development',
        logging: configService.get('NODE_ENV') === 'development',
      }),
      inject: [ConfigService],
    }),
    ThrottlerModule.forRootAsync({
      imports: [ConfigModule],
      useFactory: (configService: ConfigService) => [
        {
          ttl: configService.get('RATE_LIMIT_TTL') || 60000,
          limit: configService.get('RATE_LIMIT_MAX') || 100,
        },
      ],
      inject: [ConfigService],
    }),
    CoreModule,
    EmailModule,
    HealthModule,
  ],
  providers: [
    {
      provide: APP_GUARD,
      useClass: ThrottlerGuard,
    },
  ],
})
export class AppModule implements NestModule {
  configure(consumer: MiddlewareConsumer) {
    consumer.apply(LoggingMiddleware).forRoutes('*');
  }
}
