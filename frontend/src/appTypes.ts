import type { ReactNode } from 'react';

export type ViewKey =
  | 'overview'
  | 'knowledge'
  | 'accounts'
  | 'mediaMatrix'
  | 'campaigns'
  | 'creators'
  | 'skillPackages'
  | 'brandCompliance'
  | 'generate'
  | 'contents'
  | 'schedules'
  | 'jobs'
  | 'settings'
  | 'admin';

export type DialogKey =
  | 'knowledgeBase'
  | 'knowledgeAsset'
  | 'mediaAccount'
  | 'mediaAccountLogin'
  | 'content'
  | 'generate'
  | 'schedule'
  | 'publishPrepare'
  | null;

export type NavItem = {
  key: ViewKey;
  label: string;
  icon: ReactNode;
};
