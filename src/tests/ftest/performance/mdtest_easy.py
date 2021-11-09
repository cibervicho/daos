#!/usr/bin/python3
'''
  (C) Copyright 2019-2021 Intel Corporation.

  SPDX-License-Identifier: BSD-2-Clause-Patent
'''

from performance_test_base import PerformanceTestBase


class MdtestEasy(PerformanceTestBase):
    # pylint: disable=too-many-ancestors
    """Test class Description: Runs MdTest Easy.

    :avocado: recursive
    """

    def test_performance_mdtest_easy(self):
        """

        Test Description:
            Run MdTest Easy.

        Use Cases:
            Create a pool, container, and run MdTest Easy.

        :avocado: tags=all,full_regression
        :avocado: tags=single_node,per_rack,dragon_fly,full_system
        :avocado: tags=performance
        :avocado: tags=performance_mdtest_easy
        """
        self.run_performance_mdtest()