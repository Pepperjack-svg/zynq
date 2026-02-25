import { Injectable, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { spawn } from 'child_process';
import * as https from 'https';

const APP_VERSION = process.env.npm_package_version || '1.0.0';
const GITHUB_REPO = 'DineshMn1/zynq';

export interface VersionInfo {
  version: string;
  latest: string | null;
  hasUpdate: boolean;
}

@Injectable()
export class SystemService {
  private readonly logger = new Logger(SystemService.name);
  private cachedLatest: string | null = null;
  private cacheExpiry = 0;

  constructor(private configService: ConfigService) {}

  getVersion(): string {
    return APP_VERSION;
  }

  async checkUpdate(): Promise<VersionInfo> {
    const version = this.getVersion();
    const latest = await this.fetchLatestRelease();
    const hasUpdate = !!latest && latest !== version;
    return { version, latest, hasUpdate };
  }

  async triggerUpdate(): Promise<{ started: boolean }> {
    const appImage =
      this.configService.get<string>('APP_IMAGE') ||
      'zynqcloud/zynqcloud:latest';
    this.logger.log(`Triggering self-update — pulling ${appImage}`);

    setImmediate(() => {
      const pull = spawn('docker', ['pull', appImage], { stdio: 'inherit' });
      pull.on('close', (code) => {
        if (code !== 0) {
          this.logger.error(`docker pull exited with code ${code}`);
          return;
        }
        this.logger.log('Image pulled — restarting container');
        const containerName =
          this.configService.get<string>('CONTAINER_NAME') || 'zynqcloud';
        const restart = spawn('docker', ['restart', containerName], {
          stdio: 'inherit',
        });
        restart.on('close', (c) => {
          if (c !== 0)
            this.logger.error(`docker restart exited with code ${c}`);
        });
      });
    });

    return { started: true };
  }

  private fetchLatestRelease(): Promise<string | null> {
    const now = Date.now();
    if (this.cachedLatest !== null && now < this.cacheExpiry) {
      return Promise.resolve(this.cachedLatest);
    }

    return new Promise((resolve) => {
      const options = {
        hostname: 'api.github.com',
        path: `/repos/${GITHUB_REPO}/releases/latest`,
        headers: { 'User-Agent': 'zynqcloud-server' },
      };
      const req = https.get(options, (res) => {
        let data = '';
        res.on('data', (chunk: Buffer) => (data += chunk.toString()));
        res.on('end', () => {
          try {
            const json = JSON.parse(data) as { tag_name?: string };
            const tag = json.tag_name?.replace(/^v/, '') || null;
            this.cachedLatest = tag;
            this.cacheExpiry = Date.now() + 10 * 60 * 1000; // 10 min
            resolve(tag);
          } catch {
            resolve(null);
          }
        });
      });
      req.on('error', () => resolve(null));
      req.setTimeout(5000, () => {
        req.destroy();
        resolve(null);
      });
    });
  }
}
