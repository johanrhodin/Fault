(declare-fun multicond_t_base_value_2 () Real)
(declare-fun multicond_t_base_cond_0 () Real)
(declare-fun multicond_t_base_value_0 () Real)
(declare-fun multicond_t_base_value_1 () Real)
(assert (= multicond_t_base_cond_0 1.0))
(assert (= multicond_t_base_value_0 10.0))
(assert (= multicond_t_base_value_1 (+ multicond_t_base_value_0 20.0)))
(assert (ite (and (> multicond_t_base_cond_0 0.0) (< multicond_t_base_cond_0 4.0))(= multicond_t_base_value_2 multicond_t_base_value_1)(= multicond_t_base_value_2 multicond_t_base_value_0)))