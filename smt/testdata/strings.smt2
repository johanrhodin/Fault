(set-logic QF_NRA)
(declare-fun test_str1_0 () Bool)
(declare-fun test_str2_0 () Bool)
(declare-fun test_str3_0 () Bool)
(assert (not test_str3_0))
(assert (and test_str1_0 test_str3_0))