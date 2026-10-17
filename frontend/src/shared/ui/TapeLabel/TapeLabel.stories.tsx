import type { Meta, StoryObj } from '@storybook/react-vite';
import { TapeLabel } from './TapeLabel';

const meta = { title: 'UI/TapeLabel', component: TapeLabel } satisfies Meta<typeof TapeLabel>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Plain: Story = { args: { children: 'Album' } };
export const Accent: Story = { args: { children: 'Анонс', tone: 'accent' } };
