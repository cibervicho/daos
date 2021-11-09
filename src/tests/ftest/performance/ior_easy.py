#!/usr/bin/python3
"""
  (C) Copyright 2018-2021 Intel Corporation.

  SPDX-License-Identifier: BSD-2-Clause-Patent
"""
from performance_test_base import PerformanceTestBase

class IorEasy(PerformanceTestBase):
    # pylint: disable=too-many-ancestors
    # pylint: disable=too-few-public-methods
    """Test class Description: Run IOR Easy

    :avocado: recursive
    """

    def test_performance_ior_easy(self):
        """

        Test Description:
            Run IOR Easy

        Use Cases:
            Create a pool, container, and run IOR Easy

        :avocado: tags=all,full_regression
        :avocado: tags=single_node,per_rack,dragon_fly,full_system
        :avocado: tags=performance
        :avocado: tags=performance_ior_easy
        """
        self.run_performance_ior()