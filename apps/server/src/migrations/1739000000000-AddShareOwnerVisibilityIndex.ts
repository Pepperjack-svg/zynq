import { MigrationInterface, QueryRunner } from 'typeorm';

export class AddShareOwnerVisibilityIndex1739000000000 implements MigrationInterface {
  name = 'AddShareOwnerVisibilityIndex1739000000000';

  public async up(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(
      `CREATE INDEX IF NOT EXISTS idx_shares_created_by_is_public ON shares(created_by, is_public)`,
    );
  }

  public async down(queryRunner: QueryRunner): Promise<void> {
    await queryRunner.query(
      `DROP INDEX IF EXISTS idx_shares_created_by_is_public`,
    );
  }
}
