import type { ComponentType } from "react";
import {
  CalendarIcon,
  ClockIcon,
  DashboardIcon,
  FilmIcon,
  FolderIcon,
  HeartPulseIcon,
  MealPlanIcon,
  MusicIcon,
  SettingsIcon,
  TasksIcon,
  UtensilsIcon,
  type IconProps,
} from "./icons";

export type Page =
  | "dashboard"
  | "tasks"
  | "lists"
  | "planner"
  | "pomodoro"
  | "soundscape"
  | "recipes"
  | "mealPlan"
  | "library"
  | "health"
  | "settings";

export interface NavItem {
  id: Page;
  labelKey: string;
  icon: ComponentType<IconProps>;
}

// Primary tabs always visible on the desktop nav bar.
export const NAV_ITEMS: NavItem[] = [
  { id: "dashboard", labelKey: "nav.dashboard", icon: DashboardIcon },
  { id: "tasks", labelKey: "nav.tasks", icon: TasksIcon },
  { id: "lists", labelKey: "nav.lists", icon: FolderIcon },
  { id: "planner", labelKey: "nav.planner", icon: CalendarIcon },
  { id: "pomodoro", labelKey: "nav.timer", icon: ClockIcon },
  { id: "soundscape", labelKey: "nav.sounds", icon: MusicIcon },
  { id: "recipes", labelKey: "nav.recipes", icon: UtensilsIcon },
  { id: "mealPlan", labelKey: "nav.mealPlan", icon: MealPlanIcon },
  { id: "library", labelKey: "nav.library", icon: FilmIcon },
];

// Secondary pages tucked into the "More" dropdown so the top bar does not overflow.
export const MORE_ITEMS: NavItem[] = [
  { id: "health", labelKey: "nav.health", icon: HeartPulseIcon },
  { id: "settings", labelKey: "nav.serverSettings", icon: SettingsIcon },
];
