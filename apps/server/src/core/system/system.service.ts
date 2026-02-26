import { Injectable, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { spawn } from 'child_process';
import * as https from 'https';

const APP_VERSION = process.env.npm_package_version || '1.0.0';
const GITHUB_REPO = 'DineshMn1/zynq';

// Allowlist for docker image names: registry/namespace/name:tag
// Prevents command injection via APP_IMAGE / CONTAINER_NAME env vars.
const SAFE_IMAGE_RE = /^[a-z0-9._\-/:@]+$/i;
const SAFE_CONTAINER_RE = /^[a-zA-Z0-9_\-.]+$/;

export interface VersionInfo {
  version: string;
  latest: string | null;
  hasUpdate: boolean;
}

/** Compare two semver strings. Returns true if b > a. */
function semverGt(a: string, b: string): boolean {
  const parse = (s: string) =>
    s
      .replace(/^v/, '')
      .split('.')
      .map((n) => parseInt(n, 10) || 0);
  const [aM, am, ap] = parse(a);
  const [bM, bm, bp] = parse(b);
  if (bM !== aM) return bM > aM;
  if (bm !== am) return bm > am;
  return bp > ap;
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
    const hasUpdate = !!latest && semverGt(version, latest);
    return { version, latest, hasUpdate };
  }

  async triggerUpdate(): Promise<{
    started: boolean;
    pulled: boolean;
    restarted: boolean;
  }> {
    const appImage =
      this.configService.get<string>('APP_IMAGE') ||
      'zynqcloud/zynqcloud:latest';
    const containerName =
      this.configService.get<string>('CONTAINER_NAME') || 'zynqcloud';

    if (!SAFE_IMAGE_RE.test(appImage)) {
      this.logger.error(`Refusing to pull unsafe image name: ${appImage}`);
      return { started: false, pulled: false, restarted: false };
    }
    if (!SAFE_CONTAINER_RE.test(containerName)) {
      this.logger.error(
        `Refusing to restart unsafe container name: ${containerName}`,
      );
      return { started: false, pulled: false, restarted: false };
    }

    this.logger.log(`Triggering self-update — pulling ${appImage}`);

    const pulled = await new Promise<boolean>((resolve) => {
      const pull = spawn('docker', ['pull', appImage], { stdio: 'inherit' });
      pull.on('close', (code) => {
        if (code !== 0) {
          this.logger.error(`docker pull exited with code ${code}`);
          resolve(false);
        } else {
          resolve(true);
        }
      });
      pull.on('error', (err) => {
        this.logger.error(`docker pull error: ${err.message}`);
        resolve(false);
      });
    });

    if (!pulled) {
      return { started: true, pulled: false, restarted: false };
    }

    this.logger.log('Image pulled — restarting container');
    const restarted = await new Promise<boolean>((resolve) => {
      const restart = spawn('docker', ['restart', containerName], {
        stdio: 'inherit',
      });
      restart.on('close', (c) => {
        if (c !== 0) {
          this.logger.error(`docker restart exited with code ${c}`);
          resolve(false);
        } else {
          resolve(true);
        }
      });
      restart.on('error', (err) => {
        this.logger.error(`docker restart error: ${err.message}`);
        resolve(false);
      });
    });

    return { started: true, pulled, restarted };
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
        if (res.statusCode !== 200) {
          this.logger.warn(
            `GitHub releases API returned ${res.statusCode} — skipping update check`,
          );
          res.resume(); // drain to free socket
          resolve(null);
          return;
        }
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
