import {
  IsString,
  IsOptional,
  IsEnum,
  IsBoolean,
  IsISO8601,
  MaxLength,
  MinLength,
  Validate,
  ValidatorConstraint,
  ValidatorConstraintInterface,
} from 'class-validator';
import { SharePermission } from '../entities/share.entity';

@ValidatorConstraint({ name: 'isFutureIsoDate', async: false })
class IsFutureIsoDateConstraint implements ValidatorConstraintInterface {
  validate(value?: string): boolean {
    if (!value) {
      return true;
    }

    const timestamp = Date.parse(value);
    return !Number.isNaN(timestamp) && timestamp > Date.now();
  }

  defaultMessage(): string {
    return 'expiresAt must be a future date';
  }
}

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
  @Validate(IsFutureIsoDateConstraint)
  expiresAt?: string;

  @IsOptional()
  @IsString()
  @MinLength(6)
  @MaxLength(72)
  password?: string;
}
