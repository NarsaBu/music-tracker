import type { Meta, StoryObj } from '@storybook/react-vite';
import { LcdDisplay } from './LcdDisplay';

const meta = { title: 'UI/LcdDisplay', component: LcdDisplay } satisfies Meta<typeof LcdDisplay>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Small: Story = { args: { size: 'sm', children: 'ONLINE' } };
export const Medium: Story = { args: { size: 'md', children: '2026-10-17 06:03' } };
export const Large: Story = { args: { size: 'lg', children: '042' } };
