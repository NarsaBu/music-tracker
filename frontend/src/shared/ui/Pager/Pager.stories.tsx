import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { Pager, type PagerProps } from './Pager';

const meta = { title: 'UI/Pager', component: Pager } satisfies Meta<typeof Pager>;
export default meta;
type Story = StoryObj<typeof meta>;

function Controlled(args: PagerProps) {
  const [page, setPage] = useState(args.page);
  return <Pager {...args} page={page} onPageChange={setPage} />;
}

export const Default: Story = {
  args: { page: 6, pageCount: 15, label: 'Страницы', onPageChange: () => {} },
  render: (args) => <Controlled {...args} />,
};
