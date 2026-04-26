import { test, expect } from '@playwright/test';

test('Golden Path: Login, create invite, and view dashboard', async ({ page }) => {
  // 1. Log in
  await page.goto('/login');
  await page.fill('input[type="email"]', 'admin@example.com');
  await page.fill('input[type="password"]', 'password123');
  await page.click('button[type="submit"]');

  // Verify successful login by checking for the Dashboard header
  await expect(page.locator('h2:has-text("Dashboard")')).toBeVisible({ timeout: 10000 });

  // 2. Go to Invites page
  await page.click('text="Invites"');
  await expect(page.locator('h2:has-text("Invites")')).toBeVisible();

  // 3. Create a new Invite
  await page.click('button:has-text("Create Invite")');
  
  // Fill out the modal form
  const inviteTitle = `E2E Test Invite ${Date.now()}`;
  await page.fill('#invite-title', inviteTitle);
  await page.fill('#invite-from-at', '2030-01-01T10:00'); // Future date
  
  // Select sender (Admin User from seed data)
  await page.selectOption('#invite-from-person', { label: 'Admin User (admin@example.com)' });
  
  await page.click('button:has-text("Save")');

  // Verify the invite appears in the list
  await expect(page.locator(`text="${inviteTitle}"`)).toBeVisible();

  // 4. Return to Dashboard and verify
  await page.click('text="Dashboard"');
  await expect(page.locator('h2:has-text("Dashboard")')).toBeVisible();
  
  // E2E test complete!
});
