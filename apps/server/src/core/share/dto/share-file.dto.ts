import {
  IsString,
  IsOptional,
  IsEnum,
  IsBoolean,
  IsISO8601,
  MinLength,
} from 'class-validator';
import { SharePermission } from '../entities/share.entity';

export class ShareFileDto {
  @IsOptional()
  @IsString()
  toUserId?: string;

  @IsOptional()
  @IsString()
  email?: string;

  @IsEnum(SharePermission)
  permission: SharePermission;

  @IsOptional()
  @IsBoolean()
  isPublic?: boolean;

  @IsOptional()
  @IsISO8601()
  expiresAt?: string;

  @IsOptional()
  @IsString()
  @MinLength(6)
  password?: string;
}
