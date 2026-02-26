import { Controller, Get, Post, UseGuards } from '@nestjs/common';
import { SystemService } from './system.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { RolesGuard } from '../../common/guards/roles.guard';
import { Roles } from '../../common/decorators/roles.decorator';
import { UserRole } from '../user/entities/user.entity';

/**
 * System endpoints for version info and in-app updates.
 * The update trigger is restricted to the OWNER role only.
 */
@Controller('system')
@UseGuards(JwtAuthGuard)
export class SystemController {
  constructor(private readonly systemService: SystemService) {}

  /** Returns current version, latest GitHub release, and update flag. */
  @Get('update-check')
  async checkUpdate() {
    return this.systemService.checkUpdate();
  }

  /**
   * Triggers a docker pull + restart in the background.
   * Returns 403 for all non-OWNER roles.
   */
  @Post('update')
  @UseGuards(RolesGuard)
  @Roles(UserRole.OWNER)
  async triggerUpdate() {
    return this.systemService.triggerUpdate();
  }
}
